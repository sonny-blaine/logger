#!/usr/bin/env bash
#for i in 1 2 3 4 5
#do
nohup ab -n 3000 -c 1   -p request.json -T 'application/json' http://localhost:8383/metrics/create &
nohup ab -n 2000 -c 1 -p request400.json -T 'application/json' http://localhost:8383/metrics/create &
nohup ab -n 2000 -c 2 -p request500.json -T 'application/json' http://localhost:8383/metrics/create
#done