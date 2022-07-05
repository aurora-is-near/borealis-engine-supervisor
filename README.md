# borealis-engine-supervisor

Supervisor starts a borealis-engine instance as a subprocess and monitors it through a Prometheus server.
If the metric variable at Prometheus hasn't increased by an expected amount in the timeframe, a signal is sent to the subprocess to reset it.
In case the borealis-engine instance still isn't producing output after state reset, another signal is sent to it to terminate it.

## Running

Configure the supervisor through env variables.
For example, to start a supervisor that fetches the `engine_last_block_height_processed` metric from `http://127.0.0.1:8041` every 10 seconds and sends SIGTERM signal to the subprocess if the metric data hasn't grown by at least one, do:

```
$ export SUPERVISOR_PROMURL='http://127.0.0.1:8041'
$ export SUPERVISOR_METRIC=engine_last_block_height_processed
$ export SUPERVISOR_WARMUPDURATION=15
$ export SUPERVISOR_CHECKDURATION=10
$ export SUPERVISOR_METRICDELTA=1
$ export SUPERVISOR_FAILSIGNAL=9
$ export SUPERVISOR_HANGSIGNAL=15
$ borealis-engine-supervisor /path/to/borealis-engine [ENGINE-ARGS...]
```

All env variables of supervisor are also passed to borealis-engine.

## Configuration

| Env variable                | Meaning                                                                                   |
|-----------------------------|-------------------------------------------------------------------------------------------|
| `SUPERVISOR_PROMURL`        | Address of the prometheus metrics exporter (http://127.0.0.1:8041)                        |
| `SUPERVISOR_METRIC`         | Name of the metric to test.                                                               |
| `SUPERVISOR_WARMUPDURATION` | Seconds of warmup period after starting subprocess or state reset.                        |
| `SUPERVISOR_CHECKDURATION`  | Seconds between metric checks.                                                            |
| `SUPERVISOR_METRICDELTA`    | Expected metric delta between checks.                                                     |
| `SUPERVISOR_FAILSIGNAL`     | Signal to send if no increment of metric can be detected between first and second check.  |
| `SUPERVISOR_HANGSIGNAL`     | Signal to send if no increment of metric can be detected between i and i+1 check (i > 1). |
