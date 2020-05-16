package main

import (
	"context"
	"fmt"
	"github.com/libp2p/go-libp2p"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/routing"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	libp2pquic "github.com/libp2p/go-libp2p-quic-transport"
	secio "github.com/libp2p/go-libp2p-secio"
	libp2ptls "github.com/libp2p/go-libp2p-tls"
	"github.com/multiformats/go-multiaddr"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// Set your own keypair
	priv, _, err := crypto.GenerateKeyPair(
		crypto.Ed25519, // Select your key type. Ed25519 are nice short
		-1,             // Select key length when possible (i.e. RSA).
	)
	if err != nil {
		panic(err)
	}
	ctx := context.Background()
	var idht *dht.IpfsDHT

	node, err := libp2p.New(ctx,
		// Use the keypair we generated
		libp2p.Identity(priv),
		// Multiple listen addresses
		libp2p.ListenAddrStrings(
			"/ip4/0.0.0.0/tcp/54000", // regular tcp connections
			//"/ip4/0.0.0.0/udp/51000/quic", // a UDP endpoint for the QUIC transport
		),
		// support TLS connections
		libp2p.Security(libp2ptls.ID, libp2ptls.New),
		// support secio connections
		libp2p.Security(secio.ID, secio.New),
		// support QUIC
		libp2p.Transport(libp2pquic.NewTransport),
		// support any other default transports (TCP)
		libp2p.DefaultTransports,
		// Let's prevent our peer from having too many
		// connections by attaching a connection manager.
		libp2p.ConnectionManager(connmgr.NewConnManager(
			993,         // Lowwater
			1000,        // HighWater,
			time.Minute, // GracePeriod
		)),
		// Attempt to open ports using uPNP for NATed hosts.
		libp2p.NATPortMap(),
		// Let this host use the DHT to find other hosts

		libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) {
			idht, err = dht.New(ctx, h)
			return idht, err
		}),
		// Let this host use relays and advertise itself on relays if
		// it finds it is behind NAT. Use libp2p.Relay(options...) to
		// enable active relays and more.
		libp2p.EnableAutoRelay(),
	)
	if err != nil {
		panic(err)
	}

	log.Printf("Hello World, my second hosts ID is %s\n", node.ID())
	for _, addr := range node.Addrs() {
		log.Printf("My Addrs are: %v", addr)
	}
	var multiAddr string
	var NodeID string
	log.Printf("Enter MultiAddr")
	_, _ = fmt.Scanln(&multiAddr)
	fmt.Print(multiAddr)
	peerAddr, err := multiaddr.NewMultiaddr(multiAddr)
	if err != nil {
		log.Printf("Error in parsing multiAddress: %v", err)
	}

	log.Printf("Enter NodeID")
	_, _ = fmt.Scanln(&NodeID)
	fmt.Print(NodeID)
	peerID, err := peer.Decode(NodeID)
	if err != nil {
		log.Printf("Error decoding peer id: %v", err)
	}
	node.Network().Peerstore().AddAddr(peerID, peerAddr, time.Hour)
	conn, err := node.Network().DialPeer(context.Background(), peerID)
	if err != nil {
		log.Printf("Error in connec ting with peer: %v", err)
	} else {
		log.Printf("Remote Peer ID: %v", conn.RemotePeer())
		log.Printf("Peers: \n %v", node.Network().Peers())
	}

	// This is good for finding when we want to shutdown
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	x := <-ch
	if x == os.Signal(syscall.SIGINT) {
		log.Println("Received signal, shutting down...")
		// shut the node1 down
		if err := node.Close(); err != nil {
			panic(err)
		}
	}

}
