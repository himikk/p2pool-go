// Copyright (c) 2013-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wire

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"math"
	"math/big"
	"net"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/gertjaap/p2pool-go/logging"
)

var nullHash *chainhash.Hash

func ReadVarString(r io.Reader) (string, error) {
	len, err := ReadVarInt(r)
	if err != nil {
		return "", err
	}

	b := make([]byte, len)
	rl, err := r.Read(b)
	if rl != int(len) {
		return "", fmt.Errorf("Could not read all string bytes")
	}
	return string(b), nil
}

func ReadVarInt(r io.Reader) (uint64, error) {
	var discriminant uint8
	err := binary.Read(r, binary.LittleEndian, &discriminant)
	if err != nil {
		return 0, err
	}

	var rv uint64
	switch discriminant {
	case 0xff:
		err = binary.Read(r, binary.LittleEndian, &rv)
		if err != nil {
			return 0, err
		}

		// The encoding is not canonical if the value could have been
		// encoded using fewer bytes.
		min := uint64(0x100000000)
		if rv < min {
			return 0, fmt.Errorf("Varint not canonically packed -- uint64")
		}
	case 0xfe:
		var sv uint32
		binary.Read(r, binary.LittleEndian, &sv)
		if err != nil {
			return 0, err
		}
		rv = uint64(sv)

		// The encoding is not canonical if the value could have been
		// encoded using fewer bytes.
		min := uint64(0x10000)
		if rv < min {
			return 0, fmt.Errorf("Varint not canonically packed -- uint32")
		}
	case 0xfd:
		var sv uint16
		binary.Read(r, binary.LittleEndian, &sv)
		if err != nil {
			return 0, err
		}
		rv = uint64(sv)

		// The encoding is not canonical if the value could have been
		// encoded using fewer bytes.
		min := uint64(0xfd)
		if rv < min {
			return 0, fmt.Errorf("Varint not canonically packed -- uint16")
		}
	default:
		rv = uint64(discriminant)
	}

	return rv, nil
}

func ReadIPAddr(r io.Reader) (net.IP, error) {
	b := make([]byte, 16)
	i, err := r.Read(b)
	if i != 16 {
		return nil, fmt.Errorf("Unable to read IP address")
	}
	if err != nil {
		return nil, err
	}
	return net.IP(b), nil
}

func WriteIPAddr(w io.Writer, val net.IP) error {
	b := make([]byte, 16)
	copy(b, val.To16())
	i, err := w.Write(b)
	if i != 16 {
		return fmt.Errorf("Unable to write IP address")
	}
	return err
}

func WriteVarString(w io.Writer, val string) error {
	err := WriteVarInt(w, uint64(len(val)))
	if err != nil {
		return err
	}
	num, err := w.Write([]byte(val))
	if num != len(val) {
		return fmt.Errorf("Not all bytes could be written")
	}
	return nil
}

func WriteVarInt(w io.Writer, val uint64) error {
	if val < 0xfd {
		return binary.Write(w, binary.LittleEndian, uint8(val))
	}

	if val <= math.MaxUint16 {
		err := binary.Write(w, binary.LittleEndian, uint8(0xfd))
		if err != nil {
			return err
		}
		return binary.Write(w, binary.LittleEndian, uint16(val))
	}

	if val <= math.MaxUint32 {
		err := binary.Write(w, binary.LittleEndian, uint8(0xfe))
		if err != nil {
			return err
		}
		return binary.Write(w, binary.LittleEndian, uint32(val))
	}

	err := binary.Write(w, binary.LittleEndian, uint8(0xff))
	if err != nil {
		return err
	}
	return binary.Write(w, binary.LittleEndian, val)
}

func WriteBigInt256(w io.Writer, i *big.Int) error {
	b := make([]byte, 32)
	numBytes := i.Bytes()
	b = append(b, numBytes...)
	l, err := w.Write(b[len(b)-32:])
	if l != 32 {
		return fmt.Errorf("Couldn't write 32 bytes for big.int")
	}
	return err
}

func ReadBigInt256(r io.Reader) (*big.Int, error) {
	b := make([]byte, 32)
	i, err := r.Read(b)
	if i != 32 {
		return nil, fmt.Errorf("Couldn't read 32 bytes for big.int")
	}
	if err != nil {
		return nil, err
	}
	for b[0] == 0x00 {
		b = b[1:]
	}
	return big.NewInt(0).SetBytes(b), nil
}

func WriteChainHash(w io.Writer, i *chainhash.Hash) error {
	if i == nil {
		i = nullHash
	}
	l, err := w.Write(i.CloneBytes())
	if l != 32 {
		return fmt.Errorf("Couldn't write 32 bytes for chainhash")
	}
	return err
}

func ReadChainHash(r io.Reader) (*chainhash.Hash, error) {
	b := make([]byte, 32)
	i, err := r.Read(b)
	if i != 32 {
		return nil, fmt.Errorf("Couldn't read 32 bytes for chainhash")
	}
	if err != nil {
		return nil, err
	}
	return chainhash.NewHash(b)
}

func init() {
	nullHash, _ = chainhash.NewHashFromStr("0000000000000000000000000000000000000000000000000000000000000000")
}

func ReadSmallBlockHeader(r io.Reader) (SmallBlockHeader, error) {
	var err error
	sbh := SmallBlockHeader{}
	err = binary.Read(r, binary.LittleEndian, &sbh.Version)
	if err != nil {
		return sbh, err
	}
	sbh.PreviousBlock, err = ReadChainHash(r)
	if err != nil {
		return sbh, err
	}
	err = binary.Read(r, binary.LittleEndian, &sbh.Timestamp)
	if err != nil {
		return sbh, err
	}
	err = binary.Read(r, binary.LittleEndian, &sbh.Bits)
	if err != nil {
		return sbh, err
	}
	err = binary.Read(r, binary.LittleEndian, &sbh.Nonce)
	if err != nil {
		return sbh, err
	}
	return sbh, nil
}

func WriteSmallBlockHeader(w io.Writer, sbh SmallBlockHeader) error {
	err := binary.Write(w, binary.LittleEndian, &sbh.Version)
	if err != nil {
		return err
	}
	err = WriteChainHash(w, sbh.PreviousBlock)
	if err != nil {
		return err
	}
	err = binary.Write(w, binary.LittleEndian, &sbh.Timestamp)
	if err != nil {
		return err
	}
	err = binary.Write(w, binary.LittleEndian, &sbh.Bits)
	if err != nil {
		return err
	}
	err = binary.Write(w, binary.LittleEndian, &sbh.Nonce)
	if err != nil {
		return err
	}
	return nil
}

func WriteChainHashList(w io.Writer, list []*chainhash.Hash) error {
	err := WriteVarInt(w, uint64(len(list)))
	if err != nil {
		return err
	}

	for _, h := range list {
		WriteChainHash(w, h)
	}
	return nil
}

func ReadChainHashList(r io.Reader) ([]*chainhash.Hash, error) {
	list := make([]*chainhash.Hash, 0)
	count, err := ReadVarInt(r)
	if err != nil {
		return list, err
	}

	log.Printf("Reading chainhash list of %d elements", count)

	for i := uint64(0); i < count; i++ {
		h, err := ReadChainHash(r)
		if err != nil {
			return list, err
		}

		list = append(list, h)
	}
	return list, nil
}

func ReadSegwitData(r io.Reader) (SegwitData, error) {
	var err error
	sd := SegwitData{}

	sd.TXIDMerkleLink, err = ReadChainHashList(r)
	if err != nil {
		return sd, err
	}

	sd.WTXIDMerkleRoot, err = ReadChainHash(r)
	if err != nil {
		return sd, err
	}

	return sd, nil
}

func ReadShareData(r io.Reader) (ShareData, error) {
	var err error
	sd := ShareData{}

	sd.PreviousShareHash, err = ReadChainHash(r)
	if err != nil {
		return sd, err
	}

	sd.CoinBase, err = ReadVarString(r)
	if err != nil {
		return sd, err
	}

	err = binary.Read(r, binary.LittleEndian, &sd.Nonce)
	if err != nil {
		return sd, err
	}

	sd.PubKeyHash = make([]byte, 20)
	i, err := r.Read(sd.PubKeyHash)
	if err != nil {
		return sd, err
	}

	if i < 20 {
		return sd, fmt.Errorf("Could not read pubkeyhash. Expected 20, got %d", i)
	}

	err = binary.Read(r, binary.LittleEndian, &sd.PubKeyHashVersion)
	if err != nil {
		return sd, err
	}
	err = binary.Read(r, binary.LittleEndian, &sd.Subsidy)
	if err != nil {
		return sd, err
	}
	err = binary.Read(r, binary.LittleEndian, &sd.Donation)
	if err != nil {
		return sd, err
	}

	var staleInfo int8
	err = binary.Read(r, binary.LittleEndian, &staleInfo)
	if err != nil {
		return sd, err
	}

	sd.StaleInfo = StaleInfo(staleInfo)

	sd.DesiredVersion, err = ReadVarInt(r)
	if err != nil {
		return sd, err
	}

	return sd, nil
}

func ReadTransactionHashRefList(r io.Reader) ([]TransactionHashRef, error) {
	list := make([]TransactionHashRef, 0)
	count, err := ReadVarInt(r)
	if err != nil {
		return list, err
	}

	logging.Debugf("Reading transactionhashreflist of %d", count)
	for i := uint64(0); i < count; i++ {
		thr, err := ReadTransactionHashRef(r)
		if err != nil {
			return list, err
		}

		list = append(list, thr)
	}
	return list, nil
}

func ReadTransactionHashRef(r io.Reader) (TransactionHashRef, error) {
	var err error
	thr := TransactionHashRef{}
	thr.ShareCount, err = ReadVarInt(r)
	if err != nil {
		return thr, err
	}
	thr.TxCount, err = ReadVarInt(r)
	if err != nil {
		return thr, err
	}
	return thr, nil
}

func ReadHashLink(r io.Reader) (HashLink, error) {
	hl := HashLink{}

	stateBytes := make([]byte, 32)
	i, err := r.Read(stateBytes)
	if err != nil {
		return hl, err
	}
	if i != 32 {
		return hl, fmt.Errorf("Hashlink state not 32 bytes")
	}
	hl.State = string(stateBytes)
	log.Printf("Hashlink State: %s", hl.State)
	hl.Length, err = ReadVarInt(r)
	return hl, err
}

func ReadShareInfo(r io.Reader, segwit bool) (ShareInfo, error) {
	var err error

	si := ShareInfo{}
	si.ShareData, err = ReadShareData(r)
	if err != nil {
		return si, err
	}

	logging.Debugf("Share data read. Nonce: %d, Coinbase: %s, Subsidy: %d, Donation: %d, StaleInfo: %d ", si.ShareData.Nonce, si.ShareData.CoinBase, si.ShareData.Subsidy, si.ShareData.Donation, si.ShareData.StaleInfo)

	if segwit {
		si.SegwitData, err = ReadSegwitData(r)
		if err != nil {
			return si, err
		}
	}

	si.NewTransactionHashes, err = ReadChainHashList(r)
	if err != nil {
		return si, err
	}

	si.TransactionHashRefs, err = ReadTransactionHashRefList(r)
	if err != nil {
		return si, err
	}

	si.FarShareHash, err = ReadChainHash(r)
	if err != nil {
		return si, err
	}

	err = binary.Read(r, binary.LittleEndian, &si.MaxBits)
	if err != nil {
		return si, err
	}
	err = binary.Read(r, binary.LittleEndian, &si.Bits)
	if err != nil {
		return si, err
	}
	err = binary.Read(r, binary.LittleEndian, &si.Timestamp)
	if err != nil {
		return si, err
	}
	err = binary.Read(r, binary.LittleEndian, &si.AbsHeight)
	if err != nil {
		return si, err
	}
	absWork := make([]byte, 16) // 128 bit
	i, err := r.Read(absWork)
	if err != nil {
		return si, err
	}
	if i < 16 {
		return si, fmt.Errorf("Could not read abswork 16 bytes, read %d in stead", i)
	}
	si.AbsWork = big.NewInt(0).SetBytes(absWork)

	return si, nil
}
