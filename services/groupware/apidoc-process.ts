import * as fs from 'fs'
import * as yaml from 'js-yaml'

interface Param {
  description: string
  type: string
}

interface Config {
  params: {[param:string]:Param}
}

let inputData = ''

process.stdin.on('data', (chunk) => {
  inputData += chunk.toString()
})

process.stdin.on('end', () => {
  try {
    const config = yaml.load(fs.readFileSync('api-params.yaml', 'utf8')) as Config
    const params = config.params || {}

    const data = yaml.load(inputData) as any

    for (const path in data.paths) {
      for (const param in params) {
        if (path.includes(`{${param}}`)) {
          const paramsData = params[param] as Param
          const pathData = data.paths[path] as any
          for (const verb in pathData) {
            const verbData = pathData[verb] as any
            if (!Object.getOwnPropertyNames(verbData).includes('parameters')) {
              verbData.parameters = []
            }
            verbData['parameters'].push({
              name: param,
              required: true,
              type: paramsData.type !== 'undefined' ? paramsData.type : 'string',
              in: 'path',
              description: paramsData.description,
            })
          }
        }
      }
    }

    process.stdout.write(yaml.dump(data))
    process.stdout.write("\n")
  } catch (error) {
    if (error instanceof Error) {
      console.error(`Error occured while post-processing OpenAPI: ${error.message}`)
    } else {
      console.error("Unknown error occurred")
    }
  }
});

