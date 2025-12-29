package turn

const (
	stunMagicCookie uint32 = 0x2112A442

	// STUN message classes (RFC 5389)
	classRequest    = 0x0000
	classIndication = 0x0010
	classSuccess    = 0x0100
	classError      = 0x0110
)

const (
	// TURN methods (RFC 5766)
	methodAllocate    = 0x0003
	methodRefresh     = 0x0004
	methodSend        = 0x0006
	methodData        = 0x0007
	methodCreatePerm  = 0x0008
	methodChannelBind = 0x0009
)

const (
	// STUN attributes (partial)
	attrUsername         = 0x0006
	attrMessageIntegrity = 0x0008
	attrErrorCode        = 0x0009
	attrRealm            = 0x0014
	attrNonce            = 0x0015
	attrXorMappedAddr    = 0x0020
	attrFingerprint      = 0x8028
)

const (
	// TURN attributes (partial)
	attrRequestedTransport = 0x0019
	attrLifetime           = 0x000D
	attrXorRelayedAddr     = 0x0016
	attrXorPeerAddr        = 0x0012
	attrData               = 0x0013
	attrChannelNumber      = 0x000C
	attrEvenPort           = 0x0018
	attrReservationToken   = 0x0022
)

// TURN channel number range (RFC 5766): 0x4000-0x7FFF
const (
	channelMin = 0x4000
	channelMax = 0x7FFF
)
