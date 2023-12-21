#!/bin/bash
echo -e '# LICENSES\n\nThe following licenses are applicable to this project' > LICENSES.md
cd vendor
find . -iname 'LICENSE*' | grep -E 'LICENSE(\.(md|mit|txt|docs))?$' | sed 's|\./||' | awk '{printf("- [%s](vendor/%s)\n", $0, $0)}' >> ../LICENSES.md
