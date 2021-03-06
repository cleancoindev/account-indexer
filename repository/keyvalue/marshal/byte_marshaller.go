package marshal

import (
	"bytes"
	"errors"
	"math/big"
	"time"

	"github.com/WeTrustPlatform/account-indexer/common"
	"github.com/WeTrustPlatform/account-indexer/core/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

const (
	// TimestampByteLength length of time after marshall
	TimestampByteLength = 4
	// BlockNumberMarshallLength length of block number after marshall
	BlockNumberMarshallLength = 10
)

// ByteMarshaller marshal data using byte array
type ByteMarshaller struct {
}

// MarshallBlockValue marshall a blockIndex to []byte so that we store it as value in Block db
func (bm ByteMarshaller) MarshallBlockValue(blockIndex *types.BlockIndex) []byte {
	if blockIndex.CreatedAt == nil || blockIndex.Time == nil {
		panic(errors.New("block data is not correct"))
	}
	numAddr := len(blockIndex.Addresses)
	createdAtByteArr := common.MarshallTime(blockIndex.CreatedAt)
	timeByteArr := common.MarshallTime(blockIndex.Time)
	addrSeqLen := gethcommon.AddressLength + 1
	// time_address1_seq1_address2_seq2
	result := make([]byte, 2*TimestampByteLength+numAddr*(addrSeqLen))
	offset := 0
	// CreatedAt
	for i, byteItem := range createdAtByteArr {
		result[offset+i] = byteItem
	}
	// time
	offset = TimestampByteLength
	for i, byteItem := range timeByteArr {
		result[offset+i] = byteItem
	}
	// address_seq*
	offset = 2 * TimestampByteLength
	for i, addressSeq := range blockIndex.Addresses {
		address := addressSeq.Address
		addressByteArr, _ := hexutil.Decode(address)
		for j, byteItem := range addressByteArr {
			result[offset+i*addrSeqLen+j] = byteItem
		}
		// Last byte is the sequence
		result[offset+i*addrSeqLen+gethcommon.AddressLength] = addressSeq.Sequence
	}
	return result
}

// UnmarshallBlockValue unmarshall a byte array into array of address, this is for Block db
func (bm ByteMarshaller) UnmarshallBlockValue(value []byte) types.BlockIndex {
	addrResult := []types.AddressSequence{}
	addressSeqLen := gethcommon.AddressLength + 1
	// 4 first bytes are for CreatedAt
	createdAt := common.UnmarshallTimeToInt(value[:TimestampByteLength])
	// 4 first bytes are for time
	blockTime := common.UnmarshallTimeToInt(value[TimestampByteLength : 2*TimestampByteLength])
	numAddress := (len(value) - 2*TimestampByteLength) / (addressSeqLen)
	// remaining is for address_seq*
	addrValue := value[2*TimestampByteLength:]
	for i := 0; i < numAddress; i++ {
		address := hexutil.Encode(addrValue[i*addressSeqLen : (i+1)*addressSeqLen-1])
		sequence := addrValue[(i+1)*addressSeqLen-1]
		addressSequence := types.AddressSequence{Address: address, Sequence: sequence}
		addrResult = append(addrResult, addressSequence)
	}

	return types.BlockIndex{
		CreatedAt: createdAt,
		Time:      blockTime,
		Addresses: addrResult,
	}
}

// MarshallAddressKey create LevelDB key
func (bm ByteMarshaller) MarshallAddressKey(index *types.AddressIndex) []byte {
	return bm.MarshallAddressKeyStr(index.Address, index.Time, index.Sequence)
}

// MarshallAddressKeyStr create LevelDB key
func (bm ByteMarshaller) MarshallAddressKeyStr(address string, time *big.Int, sequence uint8) []byte {
	buf := &bytes.Buffer{}
	buf.Write(bm.MarshallAddressKeyPrefix2(address, time))
	// 1 byte for sequence
	buf.WriteByte(sequence)
	return buf.Bytes()
}

// MarshallAddressKeyPrefix marshall the address which is key prefix of address db
func (bm ByteMarshaller) MarshallAddressKeyPrefix(address string) []byte {
	resultByteArr, _ := hexutil.Decode(address)
	return resultByteArr
}

// MarshallAddressKeyPrefix2 marshall the address and time which is key prefix of address db
func (bm ByteMarshaller) MarshallAddressKeyPrefix2(address string, tm *big.Int) []byte {
	buf := &bytes.Buffer{}
	// 20 bytes
	resultByteArr, _ := hexutil.Decode(address)
	buf.Write(resultByteArr)
	// 4 byte
	timeByteArr := common.MarshallTime(tm)
	buf.Write(timeByteArr)
	return buf.Bytes()
}

// MarshallAddressKeyPrefix3 same to MarshallAddressKeyPrefix2
func (bm ByteMarshaller) MarshallAddressKeyPrefix3(address string, tm time.Time) []byte {
	buf := &bytes.Buffer{}
	// 20 bytes
	resultByteArr, _ := hexutil.Decode(address)
	buf.Write(resultByteArr)
	// 4 byte
	timeByteArr := common.MarshallTime2(tm)
	buf.Write(timeByteArr)
	return buf.Bytes()
}

// MarshallAddressValue create LevelDB value
func (bm ByteMarshaller) MarshallAddressValue(index *types.AddressIndex) []byte {
	buf := &bytes.Buffer{}
	// 32 byte
	txHashByteArr, _ := hexutil.Decode(index.TxHash)
	buf.Write(txHashByteArr)
	// 20 byte
	addressByteArr, _ := hexutil.Decode(index.CoupleAddress)
	buf.Write(addressByteArr)
	// blockNumber
	// blockNumber := blockNumberWidPad(index.BlockNumber.String())
	// buf.Write([]byte(blockNumber))
	valueByteArr := index.Value.Bytes()
	buf.Write(valueByteArr)
	return buf.Bytes()
}

// UnmarshallAddressKey LevelDB key to address_time
func (bm ByteMarshaller) UnmarshallAddressKey(key []byte) (string, *big.Int) {
	address := hexutil.Encode(key[:gethcommon.AddressLength])
	blockTime := new(big.Int)
	// TODO: should we return sequence?
	blockTime.SetBytes(key[gethcommon.AddressLength : gethcommon.AddressLength+TimestampByteLength])
	return address, blockTime
}

// UnmarshallAddressValue LevelDB value to txhash_Value_Time
func (bm ByteMarshaller) UnmarshallAddressValue(value []byte) types.AddressIndex {
	hashLength := gethcommon.HashLength
	addressLength := gethcommon.AddressLength
	prevIndex := 0
	index := hashLength
	txHash := hexutil.Encode(value[:index])
	prevIndex = index
	index = index + addressLength
	address := hexutil.Encode(value[prevIndex:index])
	// prevIndex = index
	// index = index + BlockNumberMarshallLength
	// blockNumberStr := string(value[prevIndex:index])
	// blockNumber := new(big.Int)
	// blockNumber.SetString(blockNumberStr, 10)
	prevIndex = index
	txValueBI := new(big.Int)
	txValueBI.SetBytes(value[prevIndex:])
	result := types.AddressIndex{
		TxHash:        txHash,
		CoupleAddress: address,
		Value:         txValueBI,
		// BlockNumber:   blockNumber,
	}
	return result
}

// MarshallBatchValue value of key-value init batch status database
func (bm ByteMarshaller) MarshallBatchValue(updatedAt *big.Int, currentBlock *big.Int) []byte {
	buf := &bytes.Buffer{}
	// 4 byte
	timeByteArr := common.MarshallTime(updatedAt)
	buf.Write(timeByteArr)
	blockNumberByteArr := currentBlock.Bytes()
	buf.Write(blockNumberByteArr)
	return buf.Bytes()
}

// UnmarshallBatchValue unmarshal value of key-value init batch status database
func (bm ByteMarshaller) UnmarshallBatchValue(value []byte) types.BatchStatus {
	timestamp := common.UnmarshallTimeToInt(value[:TimestampByteLength])
	currentBlock := new(big.Int)
	currentBlock.SetBytes(value[TimestampByteLength:])
	return types.BatchStatus{
		UpdatedAt: timestamp,
		Current:   currentBlock,
	}
}

// MarshallBatchKey marshall key of batch status database
func (bm ByteMarshaller) MarshallBatchKey(from *big.Int, to *big.Int, step byte, createdAt *big.Int) []byte {
	fromStr := blockNumberWidPad(from.String())
	toStr := blockNumberWidPad(to.String())
	var buffer bytes.Buffer
	buffer.WriteString(fromStr)
	buffer.WriteString(toStr)
	buffer.WriteByte(step)
	buffer.WriteString(createdAt.String())
	return buffer.Bytes()
}

// MarshallBatchKeyFrom marshall From in batch DB
func (bm ByteMarshaller) MarshallBatchKeyFrom(from *big.Int) []byte {
	fromStr := blockNumberWidPad(from.String())
	return []byte(fromStr)
}

// UnmarshallBatchKey unmarshall key of batch status database
func (bm ByteMarshaller) UnmarshallBatchKey(key []byte) types.BatchStatus {
	// TODO: handle error
	nextIndex := BlockNumberMarshallLength
	fromStr := string(key[:nextIndex])
	prevIndex := nextIndex
	nextIndex += BlockNumberMarshallLength
	toStr := string(key[prevIndex:nextIndex])
	prevIndex = nextIndex
	nextIndex++
	step := key[prevIndex]
	createdAtStr := string(key[nextIndex:])
	from := new(big.Int)
	from.SetString(fromStr, 10)
	to := new(big.Int)
	to.SetString(toStr, 10)
	createdAt := new(big.Int)
	createdAt.SetString(createdAtStr, 10)
	return types.BatchStatus{From: from, To: to, Step: step, CreatedAt: createdAt}
}

// MarshallBlockKey marshall key of block DB
func (bm ByteMarshaller) MarshallBlockKey(blockNumber string) []byte {
	return []byte(blockNumber)
}

// UnmarshallBlockKey unmarshall key of block DB
func (bm ByteMarshaller) UnmarshallBlockKey(key []byte) *big.Int {
	blockNumber := string(key)
	result := new(big.Int)
	result.SetString(blockNumber, 10)
	return result
}

func blockNumberWidPad(blockNumber string) string {
	buf := &bytes.Buffer{}
	if len(blockNumber) < BlockNumberMarshallLength {
		count := 10 - len(blockNumber)
		for i := 0; i < count; i++ {
			buf.WriteString("0")
		}
		buf.WriteString(blockNumber)
	}
	return buf.String()
}
