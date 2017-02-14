/*
 *   ESUDP - A simple udp bridge for ElasticSearch
 *   Copyright (c) 2017 Shannon Wynter.
 *
 *   This program is free software: you can redistribute it and/or modify
 *   it under the terms of the GNU General Public License as published by
 *   the Free Software Foundation, either version 3 of the License, or
 *   (at your option) any later version.
 *
 *   This program is distributed in the hope that it will be useful,
 *   but WITHOUT ANY WARRANTY; without even the implied warranty of
 *   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *   GNU General Public License for more details.
 *
 *   You should have received a copy of the GNU General Public License
 *   along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package main

import (
	"bytes"
	"flag"
	"fmt"
	"log/syslog"
	"net"
	"net/url"
	"os"
	"time"

	log "github.com/Sirupsen/logrus"
	logrus_syslog "github.com/Sirupsen/logrus/hooks/syslog"
	elastigo "github.com/mattbaird/elastigo/lib"
)

var (
	version = "Undefined"
	commit  = "Undefined"
)

var (
	defaultListen             = "0.0.0.0:9201"
	defaultUpstream           = "http://127.0.0.1:9200"
	defaultMaximumConnections = 20
	defaultMaximumRetries     = 30
	defaultBufferSize         = 10 * 1024
	defaultIndexDateFormat    = "2006-01-02"
	defaultIndexPrefix        = "app-"
	defaultSyslog             = ""
)

func main() {
	indexPrefix := flag.String("prefix", defaultIndexPrefix, "Default prefix for the index - $prefix$dateformat")
	indexDateFormat := flag.String("dateFormat", defaultIndexDateFormat, "Default date format for the index - $prefix$dateformat")
	listen := flag.String("listen", defaultListen, "UDP host:port combination to listen on")
	upstream := flag.String("upstream", defaultUpstream, "Upstream ElasticSearch")
	maximumConnections := flag.Int("maxconnections", defaultMaximumConnections, "Maximum connections to ElasticSearch")
	maximumRetries := flag.Int("maxretries", defaultMaximumRetries, "Maximum number of retries")
	maximumUDPPacket := flag.Int("maxudp", defaultBufferSize, "Maximum size of a udp packet")
	logSyslog := flag.String("syslog", defaultSyslog, "Log to remote syslog - eg localhost:514")
	showVersion := flag.Bool("version", false, "Show version and exit")

	debug := flag.Bool("debug", false, "Debug log level")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nFor help with the value of dateFormat see the golang documentation - https://golang.org/pkg/time/#Time.Format\n")
	}

	flag.Parse()

	if *showVersion {
		fmt.Printf("esudp - %s (%s)\n", version, commit)
		fmt.Println("https://github.com/freman/esudp")
		return
	}

	if *debug {
		log.SetLevel(log.DebugLevel)
	}

	if *logSyslog != "" {
		hook, err := logrus_syslog.NewSyslogHook("udp", "localhost:514", syslog.LOG_INFO, "")
		if err != nil {
			log.WithField("syslog", *logSyslog).WithError(err).Fatal("Unable to connect to syslog daemon")
		}
		log.AddHook(hook)
	}

	serverAddr, err := net.ResolveUDPAddr("udp", *listen)
	if err != nil {
		log.WithField("listen", *listen).WithError(err).Fatal("Unable to resolve listen address")
	}

	serverConn, err := net.ListenUDP("udp", serverAddr)
	if err != nil {
		log.WithFields(log.Fields{
			"listen":     *listen,
			"serverAddr": serverAddr,
		}).WithError(err).Fatal("Unable to create listening socket")
	}

	defer serverConn.Close()

	upstreamURL, err := url.Parse(*upstream)
	if err != nil {
		log.WithField("upstream", upstream).WithError(err).Fatal("Unable to parse upstream ElasticSearch URL")
	}

	client := elastigo.NewConn()
	client.Protocol = upstreamURL.Scheme
	client.Domain, client.Port, err = net.SplitHostPort(upstreamURL.Host)
	if err != nil {
		log.WithField("upstream", upstream).WithError(err).Fatal("Unable to split host and port for upstream ElasticSearch URL")
	}

	log.Debug("Creating new bulk indexer and starting it")
	indexer := client.NewBulkIndexerErrors(*maximumConnections, *maximumRetries)
	indexer.Start()

	go func() {
		for indexerError := range indexer.ErrorChannel {
			log.WithField("buffer", indexerError.Buf.String()).WithError(indexerError.Err).Error("Error while bulk indexing")
		}
	}()

	log.Debug("Bulk indexer has been started and the error channel is being read")

	buf := make([]byte, *maximumUDPPacket)
	log.Debug("Listening for UDP data")
	for {
		var rlog log.FieldLogger = log.StandardLogger()
		n, r, err := serverConn.ReadFromUDP(buf)
		if r != nil {
			rlog = rlog.WithField("remote", r.String())
		}
		if err != nil {
			rlog.WithField("read", n).WithError(err).Warn("Unable to read from UDP socket")
			continue
		}

		if n > 0 {
			rlog.WithField("buffer", buf).Debug("Got seemingly valid data from client")
			now := time.Now()

			split := bytes.SplitN(buf, []byte{':'}, 2)
			eventType := string(split[0])
			eventJSON := string(split[1])

			index := fmt.Sprintf("%s%s", *indexPrefix, now.Format(*indexDateFormat))

			if err := indexer.Index(index, eventType, "", "", "", &now, eventJSON); err != nil {
				rlog.WithFields(log.Fields{
					"index":     index,
					"event":     eventType,
					"eventJSON": eventJSON,
				}).WithError(err).Error("Unable to record event in index")
			}
		} else {
			rlog.Debug("Something is wrong, no data read?")
		}

	}
}
