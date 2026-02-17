#!/usr/bin/env python
# License: GPLv3 Copyright: 2024, Kovid Goyal <kovid at kovidgoyal.net>

import os
import subprocess


VERSION = "1.8.19"


def run(*args: str):
    cp = subprocess.run(args)
    if cp.returncode != 0:
        raise SystemExit(cp.returncode)


def main():
    version = VERSION
    try:
        ans = input(f'Publish version \033[91m{version}\033[m (y/n): ')
    except KeyboardInterrupt:
        ans = 'n'
    if ans.lower() != 'y':
        return
    os.environ['GITHUB_TOKEN'] = open(os.path.join(
        os.environ['PENV'], 'github-token')).read().strip().partition(':')[2]
    run('git', 'tag', '-a', 'v' + version, '-m', f'version {version}')
    run('git', 'push')
    run('goreleaser', 'release', '--clean')


if __name__ == '__main__':
    main()
