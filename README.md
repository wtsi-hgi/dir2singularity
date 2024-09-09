dir2singularity
===============

Adds directories to an existing singularity image as a new layer.

Usage
-----

|  Flag  |  Default     |  Description  |
|--------|--------------|---------------|
| -p     |              | path to be added to image (can be used multiple times). |
| -b     |              | path to base singularity image. |
| -o     |              | output image. |
| -r     |              | replacement prefixes, format find:replace (can be used multiple times). |
| -t     | os.TempDir() | directory to temporarily place squashfs file. |
