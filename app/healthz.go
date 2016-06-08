package app

import (
  "github.com/urfave/cli"
  "net/http"
  "net/url"
  "errors"
  "time"
  "math"
  "net"
  "fmt"
  "os"
)

func HealthzCommand() cli.Command {

  endpointFlag := cli.StringFlag{
    Name:  "endpoint, e",
    Usage: "TCP/HTTP(S) Endpoint URL, for example tcp://example:4000",
  }
  
  timeoutFlag := cli.DurationFlag{
    Name: "timeout, t",
    Usage: "Connection timeout in seconds",
    Value: 5 * time.Second,
  }

  backoffFlag := cli.Float64Flag{
    Name:  "backoff, b",
    Usage: "Rate at which to back off from retries, must be >= 1",
    Value: 1.0,
  }

  minFlag := cli.DurationFlag{
    Name:  "min, m",
    Usage: "Minimum time to wait before retrying",
    Value: 1 * time.Second,
  }

  maxFlag := cli.DurationFlag{
    Name:  "max, x",
    Usage: "Maximum time to wait before retrying",
    Value: 120 * time.Second,
  }

  return cli.Command{
    Name:  "healthz",
    Usage: "Test if an endpoint is healthy",
    Subcommands: []cli.Command{
      {
        Name: "single",
        Usage: "Test once if an endpoint is healthy",
        Action: singleHealthCheck,
        Flags: []cli.Flag{
          endpointFlag,
          timeoutFlag,
        },
      },
      {
        Name:  "loop",
        Usage: "Continuously test an endpoint until it is healthy",
        Action: loopHealthCheck,
        Flags: []cli.Flag{
          endpointFlag,
          timeoutFlag,
          backoffFlag,
          minFlag,
          maxFlag,
        },
      },
    },
    Flags: []cli.Flag{
    },
  }
}

func singleHealthCheck(c *cli.Context) error {
  if err := healthCheck(c); err != nil {
    fmt.Println(err)
    os.Exit(1)
  }
  fmt.Println("OK")
  return nil
}

func loopHealthCheck(c *cli.Context) error {
  min := c.Duration("min")
  max := c.Duration("max")
  backoff := c.Float64("backoff")
  loops := 0
  delay := min
  for err := healthCheck(c); err != nil; err = healthCheck(c) {
    fmt.Println(err)
    time.Sleep(delay)
    loops += 1
    if delay < max {
      delay = time.Duration(float64(min) * math.Pow(backoff, float64(loops)))
    }
    if delay > max {
      delay = max
    }
  }
  fmt.Println("OK")
  return nil
}

func healthCheck(c *cli.Context) error {
  timeout := c.Duration("timeout")
  url, err := url.Parse(c.String("endpoint"))
  if err != nil {
    return err
  }
  switch url.Scheme {
  case "tcp":
    var conn net.Conn
    if conn, err = net.DialTimeout(url.Scheme, url.Host, timeout); err != nil {
      return err
    }
    conn.Close()
  case "http", "https":
    client := &http.Client{
      Timeout: timeout,
    }
    var resp *http.Response
    resp, err = client.Get(c.String("endpoint"))
    if err != nil {
      return err
    }
    switch {
    case resp.StatusCode >= 200 && resp.StatusCode <= 299:
      return nil
    default:
      return errors.New(fmt.Sprintf("HTTP %d\n", resp.StatusCode))
    }
  default:
    return errors.New(fmt.Sprintf("Unsupported URL scheme: %s\n", url.Scheme))
  }
  return nil
}
