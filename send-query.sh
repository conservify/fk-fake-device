#!/bin/bash

curl -X POST --data-binary @query-files.bin http://127.0.0.1:2382/fk/v1 -v -o /dev/null

curl -X POST --data-binary @query-download-file.bin http://127.0.0.1:2382/fk/v1 -v -o /dev/null
