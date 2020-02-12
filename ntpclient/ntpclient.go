package ntpclient

import (
	"encoding/binary"
	"net"
	"time"

	"github.com/thrasher-corp/gocryptotrader/log"
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
// if no server can be reached will return local time in UTC()
func NTPClient(pool []string) time.Time {
	for i := range pool {
		con, err := net.DialTimeout("udp", pool[i], 5*time.Second)
		if err != nil {
			log.Warnf(log.TimeMgr, "Unable to connect to hosts %v attempting next", pool[i])
			continue
		}

		if err := con.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
			log.Warnf(log.TimeMgr, "Unable to SetDeadline. Error: %s\n", err)
			con.Close()
			continue
		}

		req := &ntppacket{Settings: 0x1B}
		if err := binary.Write(con, binary.BigEndian, req); err != nil {
			con.Close()
			continue
		}

		rsp := &ntppacket{}
		if err := binary.Read(con, binary.BigEndian, rsp); err != nil {
			con.Close()
			continue
		}

		secs := float64(rsp.TxTimeSec) - 2208988800
		nanos := (int64(rsp.TxTimeFrac) * 1e9) >> 32

		con.Close()
		return time.Unix(int64(secs), nanos)
	}
	log.Warnln(log.TimeMgr, "No valid NTP servers found, using current system time")
	return time.Now().UTC()
}
