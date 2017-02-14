# ESUDP

A simple udp bridge for ElasticSearch

## Usage

### -dateFormat _string_
  Default date format for the index - $prefix$dateformat (default "2006-01-02")
        
### -debug
  Debug log level
        
### -listen _string_
  UDP host:port combination to listen on (default "0.0.0.0:9201")
        
###  -maxconnections _int_
  Maximum connections to ElasticSearch (default 20)
        
###  -maxretries _int_
  Maximum number of retries (default 30)
        
###  -maxudp int
  Maximum size of a udp packet (default 10240)
        
###  -prefix string
  Default prefix for the index - $prefix$dateformat (default "app-")
        
###  -syslog string
  Log to remote syslog - eg localhost:514
        
###  -upstream string
  Upstream ElasticSearch (default "http://127.0.0.1:9200")

For help with the value of dateFormat see the golang documentation - https://golang.org/pkg/time/#Time.Format

## UDP Message Format

Super simple

```
{type}:{jsonstring}
```

Example:

```
redis:{"server":"1.2.3.4", "connect_duration":129, "data_transferred": 1280138}
```

This will create a new _redis_ record in the $prefix$dateformat index, if you wish to do any mapping of types, you should do this before you send the data for the first time.

## Building

Make sure you have the latest go installed, a properly configured _$GOPATH_ and are holding your tongue at the correct angle

```
go get -u github.com/freman/esudp
```

That's really all there is, the output binary will be in _$GOPATH_/bin

## License

Copyright (c) 2017 Shannon Wynter. Licensed under GPL3. See the [LICENSE.md](LICENSE.md) file for a copy of the license.