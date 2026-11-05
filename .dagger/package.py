"""Package the completed Nx API executable without recompiling it."""
import gzip
import hashlib
import json
from pathlib import Path
import subprocess
import tarfile


def package():
    go = lambda key: subprocess.check_output(['go', 'env', key], text=True).strip()
    platform = go('GOOS') + '/' + go('GOARCH')
    if platform not in {'linux/amd64', 'linux/arm64', 'darwin/arm64'}:
        raise ValueError('Unsupported native platform')
    binary = Path('.artifacts/bin') / platform / 'zitadel.local'
    subprocess.run([str(binary.resolve()), '--help'], check=True, stdout=subprocess.DEVNULL)
    output = Path('dist-native')
    output.mkdir(exist_ok=True)
    archive = output / ('zitadel-' + platform.replace('/', '-') + '.tar.gz')
    with archive.open('xb') as raw, gzip.GzipFile(fileobj=raw, mode='wb', filename='', mtime=0) as compressed:
        with tarfile.open(fileobj=compressed, mode='w') as bundle:
            for source, name, mode in [(binary, 'bin/zitadel', 0o755), (Path('LICENSE'), 'LICENSE', 0o644)]:
                if not source.is_file() or source.is_symlink():
                    raise ValueError('Expected a built binary and its license')
                info = tarfile.TarInfo(name)
                info.size, info.mode, info.mtime = source.stat().st_size, mode, 0
                with source.open('rb') as stream:
                    bundle.addfile(info, stream)
    with archive.open('rb') as stream:
        digest = hashlib.file_digest(stream, 'sha256').hexdigest()
    (output / 'component.json').write_text(json.dumps({'schema_version': 1, 'component': 'zitadel', 'platform': platform,
        'archive': archive.name, 'sha256': digest, 'bytes': archive.stat().st_size}, sort_keys=True) + '\n')


if __name__ == '__main__':
    package()
