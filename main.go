package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/grafov/m3u8"
)

var m3u8File = flag.String("p", "media.m3u8", "Media playist file")
var segmentsCache []*m3u8.MediaSegment

func main() {
	flag.Parse()
	f, err := os.Open(*m3u8File)
	if err != nil {
		panic(err)
	}
	p, listType, err := m3u8.DecodeFrom(bufio.NewReader(f), true)
	if err != nil {
		panic(err)
	}
	switch listType {
	case m3u8.MEDIA:
		mediapl := p.(*m3u8.MediaPlaylist)
		http.HandleFunc("/", mediaHandler(mediapl))
		http.ListenAndServe(":9080", nil)
	case m3u8.MASTER:
		masterpl := p.(*m3u8.MasterPlaylist)
		fmt.Printf("%+v\n", masterpl)
	}
}

func mediaHandler(p *m3u8.MediaPlaylist) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !p.Closed && segmentsCache == nil {
			newSegmentCache(p.Segments)
			go sliding(p)
		}

		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		io.WriteString(w, fmt.Sprintf("%+v", p))
	}
}

// Create segmentCache for sliding.
func newSegmentCache(segments []*m3u8.MediaSegment) {
	for _, seg := range segments {
		if seg == nil {
			continue
		}
		segmentsCache = append(segmentsCache, seg)
	}
}

// Removes head of chunk and append to the tail every 3 seconds.
func sliding(p *m3u8.MediaPlaylist) {
	if p.Closed {
		return
	}
	c := time.Tick(3 * time.Second)
	i := 0
	for _ = range c {
		seg := segmentsCache[i]
		slide(p, seg)
		if i == 0 {
			p.SetDiscontinuity()
		}
		i++
		if i >= len(segmentsCache) {
			i = 0
		}
	}
}

// Removes head of segments and append new to the tail.
func slide(p *m3u8.MediaPlaylist, segment *m3u8.MediaSegment) error {
	err := p.Remove()
	if err != nil {
		return err
	}
	err = p.AppendSegment(segment)
	if err != nil {
		return err
	}
	return nil
}
