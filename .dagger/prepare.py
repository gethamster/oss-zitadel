"""Keep the existing System-native loopback listener patch exact and repeatable."""
from pathlib import Path

BEFORE = 'lc.Listen(ctx, "tcp", fmt.Sprintf(":%d", port))'
AFTER = 'lc.Listen(ctx, "tcp", fmt.Sprintf("127.0.0.1:%d", port))'


def prepare(root=Path('.')):
    path = root / 'cmd/start/start.go'
    source = path.read_text()
    if source.count(AFTER) == 1 and BEFORE not in source:
        return
    if source.count(BEFORE) != 1 or AFTER in source:
        raise ValueError('The reviewed System-native listener patch no longer matches')
    path.write_text(source.replace(BEFORE, AFTER))


if __name__ == '__main__':
    prepare()
