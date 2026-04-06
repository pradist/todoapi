#!/bin/sh
unformatted=$(gofmt -l .)
if [ -n "$unformatted" ]; then
  echo "The following files are not formatted. Run: gofmt -w ."
  echo "$unformatted"
  exit 1
fi
