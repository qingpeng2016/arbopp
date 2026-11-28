package noderelay

import "time"

// PeerHealth TCP 连接与 RTT 回调（可选）。
type PeerHealth interface {
	OnPeerConnUp(remoteAddr string)
	OnPeerConnDown(remoteAddr string)
	OnPeerRTT(remoteAddr string, rtt time.Duration)
}