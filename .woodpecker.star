def main(ctx):
    return [{
        "name":"dummy-pipeline",
        "steps":[{"name":"dummy-step","image":"alpine:latest"}],
        "when":[{"branch":["main"]}],
    }]
