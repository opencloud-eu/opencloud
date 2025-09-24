import * as fs from 'fs'
import * as yaml from 'js-yaml'

const API_PARAMS_CONFIG_FILE = 'api-params.yaml'
const API_EXAMPLES_CONFIG_FILE = 'api-examples.yaml'

interface Response {
  $ref: string
}

interface Parameter {
  type: string
  required: boolean
  format: string
  example: any
  name: string
  description: string
  in: string
}

interface VerbData {
  tags: string[]
  summary: string
  description: string | undefined
  operationId: string
  parameters: Parameter[]
  responses: {[status:string]:Response}
}

interface Item {
  $ref: string
}

interface AdditionalProperties {
  $ref: string
}

interface Property {
  description: string
  type: string
  items: Item
  example: any
  additionalProperties: AdditionalProperties
}

interface Definition {
  type: string
  title: string
  required: string[]
  properties: {[property:string]:Property}
}

interface OpenApi {
  paths: {[path:string]:{[verb:string]:VerbData}}
  definitions: {[type:string]:Definition}
}

interface Param {
  description: string
  type: string
}

interface ParamsConfig {
  params: {[param:string]:Param}
}

interface ExamplesConfigExamples {
  refs: {[id:string]:any}
  inject: {[id:string]:{[property:string]:any}}
}

interface ExamplesConfig {
  examples: ExamplesConfigExamples
}

let inputData = ''

process.stdin.on('data', (chunk) => {
  inputData += chunk.toString()
})

const usedExamples = new Set<string>()
const unresolvedExampleReferences = new Set<string>()

process.stdin.on('end', () => {
  try {
    const paramsConfig = yaml.load(fs.readFileSync(API_PARAMS_CONFIG_FILE, 'utf8')) as ParamsConfig
    const params = paramsConfig.params || {}

    const examplesConfig = yaml.load(fs.readFileSync(API_EXAMPLES_CONFIG_FILE, 'utf8')) as ExamplesConfig
    const exampleRefs = examplesConfig.examples.refs
    const exampleInjects = examplesConfig.examples.inject

    const data = yaml.load(inputData) as OpenApi

    for (const path in data.paths) {
      const pathData = data.paths[path]

      for (const param in params) {
        if (path.includes(`{${param}}`)) {
          const paramsData = params[param] as Param
          for (const verb in pathData) {
            const verbData = pathData[verb]
            verbData.parameters ??= []
            verbData.parameters.push({
              name: param,
              required: true,
              type: paramsData.type !== undefined ? paramsData.type : 'string',
              in: 'path',
              description: paramsData.description,
            } as Parameter)
          }
        }
      }

      // do some magic with the formatting of endpoint descriptions:
      for (const verb in pathData) {
        const verbData = pathData[verb]
        if (verbData.description !== null && verbData.description !== undefined) {
          verbData.description = verbData.description.split("\n").map((line) => {
            return line.replace(/^(\s*)!/, '$1*')
          }).join("\n")
        }
      }
    }

    for (const def in data.definitions) {
      const defData = data.definitions[def]
      if (defData.properties !== null && defData.properties !== undefined) {
        const injects = exampleInjects[def] || {}
        for (const prop in defData.properties as any) {
          const propData = defData.properties[prop]

          const inject = injects[prop]
          if (inject !== null && inject !== undefined) {
            propData.example = inject
          }

          if (propData.example !== null && propData.example !== undefined) {
            if (typeof propData.example === 'string' && (propData.example as string).startsWith('$')) {
              const exampleId = propData.example.substring(1)
              const value = exampleRefs[exampleId]
              if (value === null || value === undefined) {
                unresolvedExampleReferences.add(exampleId)
              } else {
                usedExamples.add(exampleId)
                propData.example = value
              }
            }
          }
        }
      }
    }

    process.stdout.write(yaml.dump(data))
    process.stdout.write("\n")

    if (unresolvedExampleReferences.size > 0) {
      console.error(`\x1b[33;1m⚠️ WARNING: unresolved example references not contained in ${API_PARAMS_CONFIG_FILE}:\x1b[0m`)
      unresolvedExampleReferences.forEach(item => {
        console.error(`  - ${item}`)
      })
      console.error()
    }

    const unusedExampleReferences = new Set<string>(Object.keys(exampleRefs))
    usedExamples.forEach(item => {
      unusedExampleReferences.delete(item)
    })

    if (unusedExampleReferences.size > 0) {
      console.error(`\x1b[33;1m⚠️ WARNING: unused examples in ${API_EXAMPLES_CONFIG_FILE}:\x1b[0m`)
      unusedExampleReferences.forEach(item => {
        console.error(`  - ${item}`)
      })
      console.error()
    }

  } catch (error) {
    if (error instanceof Error) {
      console.error(`Error occured while post-processing OpenAPI: ${error.message}`)
    } else {
      console.error("Unknown error occurred")
    }
  }
});

