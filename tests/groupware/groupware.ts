import { SharedArray } from 'k6/data'
import http from 'k6/http'
import encoding from 'k6/encoding'
import exec from 'k6/execution'

import { randomItem } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js'
import papaparse from 'https://jslib.k6.io/papaparse/5.1.1/index.js'
import { URL } from 'https://jslib.k6.io/url/1.0.0/index.js'
import { check, fail, group } from 'k6'
import { Counter } from 'k6/metrics'

export const options = {
  noConnectionReuse: true,
  noVUConnectionReuse: true,
  insecureSkipTLSVerify: true,
  scenarios: {
    rampup: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { target: 50, duration: '30s' },
        { target: 75, duration: '30s' },
        { target: 100, duration: '60s' },
        { target: 50, duration: '20s' },
      ],
      gracefulRampDown: '10s',
    },
  },
}

const TEST_USER_NAMES: string|undefined = __ENV.TEST_USER_NAMES
const TEST_USER_PASSWORD: string = __ENV.TEST_USER_PASSWORD ?? 'demo'
const TEST_USER_DOMAIN: string = __ENV.TEST_USER_DOMAIN ?? 'example.org'
const CLOUD_URL: string = __ENV.CLOUD_URL ?? 'https://cloud.opencloud.test'
const KEYCLOAK_URL: string = __ENV.KEYCLOAK_URL ?? 'https://keycloak.opencloud.test/realms/openCloud'
const KEYCLOAK_CLIENT_ID: string = __ENV.KEYCLOAK_CLIENT_ID ?? 'groupware'
const USERS_FILE: string = __ENV.USERS_FILE ?? 'users.csv'
const JWT_EXPIRATION_THRESHOLD_SECONDS: number = parseInt(__ENV.JWT_EXPIRATION_THRESHOLD_SECONDS ?? '2')

type JwtHeader = {
  alg: string
  typ: string
  kid: string
}

type JwtPayload = {
  exp: number
  iat: number
}

type Jwt = {
  header: JwtHeader
  payload: JwtPayload
  signature: string
}

function decodeJwt(token: string): Jwt {
    const parts = token.split('.')
    const header = JSON.parse(encoding.b64decode(parts[0], 'rawurl', 's')) as JwtHeader
    const payload = JSON.parse(encoding.b64decode(parts[1], 'rawurl', 's')) as JwtPayload
    const signature = encoding.b64decode(parts[2], 'rawurl', 's')
    return {header: header, payload: payload, signature: signature} as Jwt
}

type User = {
  name: string
  password: string
  mail: string
}

type Identity = {
  id: string
  name: string
  email: string
  replyTo: string | undefined
  bcc: string | undefined
  textSignature: string | undefined
  htmlSignature: string | undefined
  mayDelete: boolean
}

type IdentityGetResponse = {
  accountId: string
  state: string
  list: Identity[]
  notFound: string[] | undefined
}

type VacationResponseGetResponse = {
  accountId: string
  state: string
  notFound: string[]
}

type EmailAddress = {
  name: string | undefined
  address: string
}

type Message = {
  '@odata.etag': string
  id: string
  createdDateTime: string
  receivedDateTime: string
  sentDateTime: string
  internetMessageId: string
  subject: string
  bodyPreview: string
  from: EmailAddress | undefined
  toRecipients: EmailAddress[]
  ccRecipients: EmailAddress[]
  parentFolderId: string
  conversationId: string
  webLink: string
}

type Messages = {
  '@odata.context': string
  value: Message[]
}

function token(user: User): string {
  const res = http.post(`${KEYCLOAK_URL}/protocol/openid-connect/token`, {
    client_id: KEYCLOAK_CLIENT_ID,
    scope: 'openid',
    grant_type: 'password',
    username: user.name,
    password: user.password,
  })
  if (res.status !== 200) {
    fail(`failed to retrieve token for ${user.name}: ${res.status} ${res.status_text}`)
  }
  const accessToken = res.json('access_token')?.toString()
  if (accessToken === undefined) {
    fail(`access token is empty for ${user.name}`)
  } else {
    return accessToken
  }
}

function authenticate(user: User): Auth {
  const raw = token(user)
  const jwt = decodeJwt(raw)
  return {raw: raw, jwt: jwt} as Auth
}

const users: User[] = new SharedArray('users', function () {
  if (TEST_USER_NAMES) {
    return TEST_USER_NAMES.split(',').map((name) => { return {name: name, password: TEST_USER_PASSWORD, mail: `${name}@${TEST_USER_DOMAIN}`} as User })
  } else {
    return papaparse.parse(open(USERS_FILE), { header: true, skipEmptyLines: true, }).data.map((row:object) => row as User)
  }
})

type Auth = {
  raw: string
  jwt: Jwt
}

type TestData = {
  auth: object
}

export function setup(): TestData {
  const auth = {}
  for (const user of users) {
    const a = authenticate(user)
    auth[user.name] = a
  }
  return {
    auth: auth,
  } as TestData
}

const stalwartIdRegex = /^[0-9a-z]+$/

export default function testSuite(data: TestData) {
  const user = randomItem(users) as User
  let auth = data.auth[user.name]

  if (auth === undefined) {
    fail(`missing authentication for user ${user.name}`)
  }
  const now = Math.floor(Date.now() / 1000)
  if (auth.jwt.payload.exp - now < JWT_EXPIRATION_THRESHOLD_SECONDS) {
    exec.test.abort(`token is expired for ${user.name}, need to renew`)
  }

  group('retrieve user identity using /me/identity', () => {
    const res = http.get(`${CLOUD_URL}/graph/v1.0/me/identity`, {headers: {Authorization: `Bearer ${auth.raw}`}})
    check(res, {
      'is status 200': (r) => r.status === 200,
    });

    const response = res.json() as IdentityGetResponse
    check(response, {
      'identity response has an accountId': r => r.accountId !== undefined && stalwartIdRegex.test(r.accountId),
      'identity response has a state': r => r.state !== undefined && stalwartIdRegex.test(r.state),
      'identity response has an empty notFound': r => r.notFound === undefined,
      'identity response has one identity item in its list': r => r.list && r.list.length === 1,
      'identity response has one identity item with an id': r => r.list && r.list.length === 1 && stalwartIdRegex.test(r.list[0].id),
      'identity response has one identity item with a name': r => r.list && r.list.length === 1 && r.list[0].name !== undefined,
      'identity response has one identity item with the expected email': r => r.list && r.list.length === 1 && r.list[0].email === user.mail,
      'identity response has one identity item with mayDelete=true': r => r.list && r.list.length === 1 && r.list[0].mayDelete === true,
      'identity response has one identity item with an empty replyTo': r => r.list && r.list.length === 1 && r.list[0].replyTo === undefined,
      'identity response has one identity item with an empty bcc': r => r.list && r.list.length === 1 && r.list[0].bcc === undefined,
      'identity response has one identity item with an empty textSignature': r => r.list && r.list.length === 1 && r.list[0].textSignature === undefined,
      'identity response has one identity item with an empty htmlSignature': r => r.list && r.list.length === 1 && r.list[0].htmlSignature === undefined,
    })
  })

  group('retrieve user vacationresponse using /me/vacation', () => {
    const res = http.get(`${CLOUD_URL}/graph/v1.0/me/vacation`, {headers: {Authorization: `Bearer ${auth.raw}`}})
    check(res, {
      'is status 200': (r) => r.status === 200,
    });

    const response = res.json() as VacationResponseGetResponse
    check(response, {
      'vacation response has an accountId': r => r.accountId !== undefined && stalwartIdRegex.test(r.accountId),
      'vacation response has a state': r => r.state !== undefined && stalwartIdRegex.test(r.state),
      'vacation response has a notFound that only contains "singleton"': r => r.notFound && r.notFound.length == 1 && r.notFound[0] == 'singleton',
    })
  })

  group('retrieve user top message using /me/messages', () => {
    const url = new URL(`${CLOUD_URL}/graph/v1.0/me/messages`)
    url.searchParams.append('$top', '1')
    const res = http.get(url.toString(), {headers: {Authorization: `Bearer ${auth.raw}`}})
    check(res, {
      'is status 200': (r) => r.status === 200,
    });

    const response = res.json() as Messages
    check(response, {
      'messages has a context': r => r['@odata.context'] !== undefined,
      'messages has a value with a length of 0 or 1': r => r.value !== undefined && (r.value.length === 0 || r.value.length === 1),
      'if there is a message, it has a subject': r => r.value !== undefined && (r.value.length === 0 || r.value[0].subject !== ''),
    })
  })
}
