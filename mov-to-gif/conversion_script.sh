#!/bin/sh

set -e

# Credit to this SO answer by alexey-kozhevnikov: https://superuser.com/a/836349
ffmpeg -i /content/input -vf 'fps=5, scale=if(gte(iw\,ih)\,min(1280\,iw)\,-2):if(lt(iw\,ih)\,min(1280\,ih)\,-2)' /tmp/output.gif
# Credit to this SO answer by pleasestand: https://superuser.com/a/436109
convert -layers Optimize /tmp/output.gif /content/output_optimized.gif

