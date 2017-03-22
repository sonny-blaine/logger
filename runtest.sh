#!/usr/bin/env bash

#200
echo "Start Requests With status 200" &
nohup ab -n 1000 -c 3 -p reqs/200/agoracred.json -T 'application/json' http://localhost:8383/metrics/create &
nohup ab -n 1000 -c 3 -p reqs/200/rovereti.json -T 'application/json' http://localhost:8383/metrics/create &
nohup ab -n 1000 -c 3 -p reqs/200/vtex.json -T 'application/json' http://localhost:8383/metrics/create &

#400
echo "Start Requests With status 400" &
nohup ab -n 1000 -c 2 -p reqs/400/agoracred.json -T 'application/json' http://localhost:8383/metrics/create &
nohup ab -n 1000 -c 2 -p reqs/400/rovereti.json -T 'application/json' http://localhost:8383/metrics/create &
nohup ab -n 1000 -c 2 -p reqs/400/vtex.json -T 'application/json' http://localhost:8383/metrics/create &

#500
echo "Start Requests With status 500" &
nohup ab -n 1000 -c 1 -p reqs/500/agoracred.json -T 'application/json' http://localhost:8383/metrics/create &
nohup ab -n 1000 -c 1 -p reqs/500/rovereti.json -T 'application/json' http://localhost:8383/metrics/create &
nohup ab -n 1000 -c 1 -p reqs/500/vtex.json -T 'application/json' http://localhost:8383/metrics/create