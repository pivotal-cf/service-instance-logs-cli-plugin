#!/bin/bash

if [ ! -z ${DEBUG} ]; then
    set -x
fi

declare -a SCS_COMMANDS=("service-instance-logs")
CMD_DOC_FILENAME=cli.md

echo "# Service Instance Logs CF CLI Plugin Docs

The following commands can be executed using the Service Instance Logs [Cloud Foundry CLI](https://github.com/cloudfoundry/cli) Plugin.

# Service Instance Logs CLI Docs

" > $CMD_DOC_FILENAME

for i in "${SCS_COMMANDS[@]}"
do
    echo "Capturing help documentation from `cf $i` command"
    echo "## \`cf $i\`

\`\`\`" >> $CMD_DOC_FILENAME
    
    cf help $i >> $CMD_DOC_FILENAME
    
    echo "\`\`\`

" >> $CMD_DOC_FILENAME
done

echo "Print contents of $CMD_DOC_FILENAME"
echo "==================================="
cat $CMD_DOC_FILENAME