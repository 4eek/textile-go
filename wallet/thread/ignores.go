package thread

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/segmentio/ksuid"
	"github.com/textileio/textile-go/pb"
	"github.com/textileio/textile-go/repo"
	"gx/ipfs/QmZoWKhxUmZ2seW4BzX6fJkNR8hh9PsGModr7q171yq2SS/go-libp2p-peer"
	mh "gx/ipfs/QmZyZDi491cCNTLfAhwcaDii2Kg4pwKRkhqQzURGDvY6ua/go-multihash"
	libp2pc "gx/ipfs/QmaPbCnUMBohSGo3KnxEa2bHqyJVVeEEcwtqJAYxerieBo/go-libp2p-crypto"
	"time"
)

// Ignore adds an outgoing ignore block, dataId is the target block to ignore
func (t *Thread) Ignore(dataId string) (mh.Multihash, error) {
	t.mux.Lock()
	defer t.mux.Unlock()

	// dataId is a fellow block id,
	// adding an ignore specific prefix here to ensure future flexibility
	dataId = fmt.Sprintf("ignore-%s", dataId)

	// build block
	header, err := t.newBlockHeader(time.Now())
	if err != nil {
		return nil, err
	}
	content := &pb.ThreadIgnore{
		Header: header,
		DataId: dataId,
	}

	// commit to ipfs
	message, addr, err := t.commitBlock(content, pb.Message_THREAD_IGNORE)
	if err != nil {
		return nil, err
	}
	id := addr.B58String()

	// index it locally
	dconf := &repo.DataBlockConfig{
		DataId: dataId,
	}
	if err := t.indexBlock(id, header, repo.IgnoreBlock, dconf); err != nil {
		return nil, err
	}

	// update head
	if err := t.updateHead(id); err != nil {
		return nil, err
	}

	// post it
	t.post(message, id, t.Peers())

	log.Debugf("ignore added to %s: %s", t.Id, id)

	// all done
	return addr, nil
}

// HandleIgnoreBlock handles an incoming ignore block
func (t *Thread) HandleIgnoreBlock(message *pb.Envelope, signed *pb.SignedThreadBlock, content *pb.ThreadIgnore, following bool) (mh.Multihash, error) {
	// unmarshal if needed
	if content == nil {
		content = new(pb.ThreadIgnore)
		if err := proto.Unmarshal(signed.Block, content); err != nil {
			return nil, err
		}
	}

	// add to ipfs
	addr, err := t.addBlock(message)
	if err != nil {
		return nil, err
	}
	id := addr.B58String()

	// check if we aleady have this block indexed
	// (should only happen if a misbehaving peer keeps sending the same block)
	index := t.blocks().Get(id)
	if index != nil {
		return nil, nil
	}

	// get the author id
	authorPk, err := libp2pc.UnmarshalPublicKey(content.Header.AuthorPk)
	if err != nil {
		return nil, err
	}
	authorId, err := peer.IDFromPublicKey(authorPk)
	if err != nil {
		return nil, err
	}

	// add author as a new local peer, just in case we haven't found this peer yet.
	// double-check not self in case we're re-discovering the thread
	if authorId.Pretty() != t.ipfs().Identity.Pretty() {
		newPeer := &repo.Peer{
			Row:      ksuid.New().String(),
			Id:       authorId.Pretty(),
			ThreadId: libp2pc.ConfigEncodeKey(content.Header.ThreadPk),
			PubKey:   content.Header.AuthorPk,
		}
		if err := t.peers().Add(newPeer); err != nil {
			// TODO: #202 (Properly handle database/sql errors)
			log.Warningf("peer with id %s already exists in thread %s", newPeer.Id, t.Id)
		}
	}

	// index it locally
	dconf := &repo.DataBlockConfig{
		DataId: content.DataId,
	}
	if err := t.indexBlock(id, content.Header, repo.IgnoreBlock, dconf); err != nil {
		return nil, err
	}

	// back prop
	if err := t.FollowParents(content.Header.Parents); err != nil {
		return nil, err
	}

	// handle HEAD
	if following {
		return addr, nil
	}
	if _, err := t.handleHead(id, content.Header.Parents, false); err != nil {
		return nil, err
	}

	return addr, nil
}
