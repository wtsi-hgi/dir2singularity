dir2singularity
===============

Adds directories to an existing singularity image as a new layer.

Usage
-----

|  Flag  |  Default     |  Description  |
|--------|--------------|---------------|
| -b     |              | path to base singularity image. |
| -e     |              | env var to add to the singularity image (can be used multiple times). |
| -o     |              | output image. |
| -p     |              | path to be added to image (can be used multiple times). |
| -r     |              | replacement prefixes, format find:replace (can be used multiple times). |
| -t     | os.TempDir() | directory to temporarily place squashfs file. |
