# httptester
A program to test the behavior of HTTP proxies.

## Getting Started
You can install httptester as follows:

```
go install github.com/ema/httptester
```

Write a test file such as the following:

```
# Test a basic get request

handle "/endpoint/1" {
    expect req.method eq "GET"
    expect req.headers["User-Agent"] ~ "chrome"
    expect req.body eq ""
    tx -body "Hello world!" -header "X-HTC-Origin: true" -status 200
}

client "nemo" {
    tx -url "/endpoint/1" -method "GET" -header "User-Agent: this might look like chrome to some"
    expect resp.status ne 404
    expect resp.headers["Server"] ~ "^ATS/[0-9]\.[0-9]\.[0-9]$"
    expect resp.headers["Something-That-Should-Not-Be-Set"] eq ""
    expect resp.status eq 200
}
```

Save the test somewhere, for example as **get.htc**. Then run it with:

```
$ httptester -verbose get.htc
2020/06/25 16:58:27 Finished waiting for http://localhost:45521/httpTesterInternalCheck
2020/06/25 16:58:30 Finished waiting for http://localhost:34965/httpTesterInternalCheck
2020/06/25 16:58:30 Proxy started using temporary directory /tmp/runroot414268077
2020/06/25 16:58:30 Sending GET /endpoint/1
User-Agent:  this might look like chrome to some

2020/06/25 16:58:30 Exiting in 0 seconds
$ echo $?
0
```
Try to change some of the **expect**, for instance expecting the request to be
a POST instead, and verify that the test fails:

```
$ httptester not-a-post.htc
2020/06/25 16:59:17 FAILED: "req.method eq \"POST\"" (actual="GET")
$ echo $?
1
```
