// Small k6 script to generate some load on read-only endpoints of
// the Graph API, for showcasing the metrics.

import http from 'k6/http';
import { check, sleep } from 'k6';
import encoding from 'k6/encoding';

// Configuration via environment variables with defaults
const BASE_URL = __ENV.BASE_URL || 'https://localhost:9200';
const USERNAME = __ENV.USERNAME || 'alan';
const PASSWORD = __ENV.PASSWORD || 'demo';

export const options = {
  insecureSkipTLSVerify: true,
  vus: 10,
  thresholds: {
    http_req_failed: ['rate<0.01'],
    http_req_duration: ['p(95)<500'],
  },
};

const credentials = `${USERNAME}:${PASSWORD}`;
const encodedCredentials = encoding.b64encode(credentials);

const params = {
  headers: {
    'Authorization': `Basic ${encodedCredentials}`,
    'Accept': 'application/json',
  },
};

export default function () {
  // Fetch current user profile, including the list of groups the user is part of
  let resMe = http.get(`${BASE_URL}/graph/v1.0/me?$expand=memberOf`, params);
  const meOk = check(resMe, { 'GET /me status is 200': (r) => r.status === 200 });
  sleep(0.1);
  // extract the names of the groups the user is part of, because the user is allowed
  // to retrieve information about those
  let groupNames = [];
  if (meOk && resMe.json() && resMe.json().memberOf) {
    groupNames = (resMe.json().memberOf || []).map((group) => group.displayName);
  }

  // Fetch oneself using the users search API:
  let resUsers = http.get(`${BASE_URL}/graph/v1.0/users?$search="${USERNAME}"`, params);
  check(resUsers, { 'GET /users status is 200': (r) => r.status === 200 });
  sleep(0.1);

  // Fetch storage drives
  let resDrives = http.get(`${BASE_URL}/graph/v1.0/drives`, params);
  const drivesOk = check(resDrives, {
    'GET /drives status is 200': (r) => r.status === 200,
  });
  sleep(0.1);

  // For each of those drives, retrieve deeper information about each
  if (drivesOk && resDrives.json() && resDrives.json().value) {
    const drives = resDrives.json().value;

    if (drives.length > 0) {
      const driveId = drives[0].id;
      let resDrive = http.get(`${BASE_URL}/graph/v1.0/drives/${driveId}`, params);

      check(resDrive, {
        'GET /drives/{id} status is 200': (r) => r.status === 200,
      });
    }
  }

  // For each of the groups the user is part of, retrieve information about each of them
  // using the group searching endpoint
  for (const group of groupNames) {
    let resGroups = http.get(`${BASE_URL}/graph/v1.0/groups?$search="${group}"`, params);
    const groupsOk = check(resGroups, {
      'GET /groups status is 200': (r) => r.status === 200,
    });
  }

  sleep(0.2);
}
