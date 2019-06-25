package ntpclient

import (
	"encoding/binary"
	"errors"
	"net"
	"time"

	log "github.com/idoall/gocryptotrader/logger"
)

type ntppacket struct {
	Settings       uint8  // leap yr indicator, ver number, and mode
	Stratum        uint8  // stratum of local clock
	Poll           int8   // poll exponent
	Precision      int8   // precision exponent
	RootDelay      uint32 // root delay
	RootDispersion uint32 // root dispersion
	ReferenceID    uint32 // reference id
	RefTimeSec     uint32 // reference timestamp sec
	RefTimeFrac    uint32 // reference timestamp fractional
	OrigTimeSec    uint32 // origin time secs
	OrigTimeFrac   uint32 // origin time fractional
	RxTimeSec      uint32 // receive time secs
	RxTimeFrac     uint32 // receive time frac
	TxTimeSec      uint32 // transmit time secs
	TxTimeFrac     uint32 // transmit time frac
}

// NTPClient create's a new NTPClient and returns local based on ntp servers provided timestamp
func NTPClient(pool []string) (time.Time, error) {
	for i := range pool {
		con, err := net.Dial("udp", pool[i])
		if err != nil {
			log.Warnf("Unable to connect to hosts %v attempting next", pool[i])
			continue
		}

		defer con.Close()

		con.SetDeadline(time.Now().Add(5 * time.Second))

		req := &ntppacket{Settings: 0x1B}
		if err := binary.Write(con, binary.BigEndian, req); err != nil {
			continue
		}

		rsp := &ntppacket{}
		if err := binary.Read(con, binary.BigEndian, rsp); err != nil {
			continue
		}

		secs := float64(rsp.TxTimeSec) - 2208988800
		nanos := (int64(rsp.TxTimeFrac) * 1e9) >> 32

		return time.Unix(int64(secs), nanos), nil
	}
	return time.Unix(0, 0), errors.New("no valid time servers")
}
