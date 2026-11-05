"""Run the repository's existing Nx API build through the managed controller."""
from typing import Annotated
import dagger
from dagger import DefaultPath, dag, function, object_type

GO = 'golang@sha256:23ce71337b726608afb1bc00986e630a05034a87a87ea1fc2cd1ae5cf33387eb'
NODE = 'node@sha256:2f0e17f8955e766e776ae16d3ac47519ca3edf47c07e5c1423527926fdd5d568'


@object_type
class ZitadelBuild:
    def container(self, source, platform):
        if platform not in {'linux/amd64', 'linux/arm64'}:
            raise ValueError('macOS uses the enrolled native worker')
        node = dag.container(platform=dagger.Platform(platform)).from_(NODE)
        return (dag.container(platform=dagger.Platform(platform)).from_(GO)
                .with_exec(['sh', '-ec', 'apt-get update && apt-get install -y --no-install-recommends python3 && rm -rf /var/lib/apt/lists/*'])
                .with_directory('/usr/local', node.directory('/usr/local'))
                .with_exec(['corepack', 'enable', 'pnpm'])
                .with_exec(['sh', '-ec', 'useradd --uid 1000 --create-home builder'])
                .with_env_variable('HOME', '/home/builder')
                .with_env_variable('PATH', '/usr/local/go/bin:/usr/local/bin:/usr/bin:/bin')
                .with_env_variable('GOTOOLCHAIN', 'go1.26.10').with_env_variable('GOFLAGS', '-p=2')
                .with_env_variable('GOMAXPROCS', '2').with_env_variable('NODE_OPTIONS', '--max-old-space-size=4096')
                .with_env_variable('PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD', '1').with_env_variable('CYPRESS_INSTALL_BINARY', '0')
                .with_env_variable('NX_DAEMON', 'false').with_env_variable('NX_NO_CLOUD', 'true')
                .with_mounted_cache('/go/pkg/mod', dag.cache_volume('zitadel-go-modules'), owner='1000:1000')
                .with_mounted_cache('/cache/go-build', dag.cache_volume('zitadel-go-' + platform), owner='1000:1000')
                .with_env_variable('GOCACHE', '/cache/go-build')
                .with_mounted_cache('/pnpm', dag.cache_volume('zitadel-pnpm'), owner='1000:1000')
                .with_directory('/source', source, exclude=['.git', '.artifacts', '.nx', 'node_modules'], owner='1000:1000')
                .with_exec(['chown', '-R', '1000:1000', '/go', '/cache/go-build', '/pnpm'])
                .with_user('1000:1000')
                .with_workdir('/source')
                .with_exec(['python3', '.dagger/prepare.py'])
                .with_exec(['pnpm', 'install', '--frozen-lockfile', '--store-dir', '/pnpm']))

    @function
    async def test(self, platform: str, source: Annotated[dagger.Directory, DefaultPath('/')], variables: str = '{}') -> str:
        if variables != '{}':
            raise ValueError('This recipe has no build-variable overrides')
        return await self.container(source, platform).with_exec(['pnpm', 'nx', 'run', '@zitadel/api:test-unit']).stdout()

    @function
    def build(self, platform: str, source: Annotated[dagger.Directory, DefaultPath('/')], variables: str = '{}') -> dagger.Directory:
        if variables != '{}':
            raise ValueError('This recipe has no build-variable overrides')
        return (self.container(source, platform)
                .with_exec(['pnpm', 'nx', 'run', '@zitadel/api:build'])
                .with_exec(['python3', '.dagger/package.py'])
                .directory('/source/dist-native'))
