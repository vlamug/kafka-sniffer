package kafka

import "fmt"

const (
	unknownRecords = iota
	legacyRecords
	defaultRecords

	magicOffset = 16
	magicLength = 1
)

// Records implements a union type containing either a RecordBatch or a legacy MessageSet.
type Records struct {
	recordsType int
	MsgSet      *MessageSet
	RecordBatch *RecordBatch
}

// setTypeFromFields sets type of Records depending on which of MsgSet or RecordBatch is not nil.
// The first return value indicates whether both fields are nil (and the type is not set).
// If both fields are not nil, it returns an error.
func (r *Records) setTypeFromFields() (bool, error) {
	if r.MsgSet == nil && r.RecordBatch == nil {
		return true, nil
	}
	if r.MsgSet != nil && r.RecordBatch != nil {
		return false, fmt.Errorf("both MsgSet and RecordBatch are set, but record type is unknown")
	}
	r.recordsType = defaultRecords
	if r.MsgSet != nil {
		r.recordsType = legacyRecords
	}
	return false, nil
}


func (r *Records) setTypeFromMagic(pd PacketDecoder) error {
	magic, err := magicValue(pd)
	if err != nil {
		return err
	}

	r.recordsType = defaultRecords
	if magic < 2 {
		r.recordsType = legacyRecords
	}

	return nil
}

func (r *Records) Decode(pd PacketDecoder) error {
	if r.recordsType == unknownRecords {
		if err := r.setTypeFromMagic(pd); err != nil {
			return err
		}
	}

	switch r.recordsType {
	case legacyRecords:
		r.MsgSet = &MessageSet{}
		return r.MsgSet.Decode(pd)
	case defaultRecords:
		r.RecordBatch = &RecordBatch{}
		return r.RecordBatch.Decode(pd)
	}
	return fmt.Errorf("unknown records type: %v", r.recordsType)
}

func (r *Records) numRecords() (int, error) {
	if r.recordsType == unknownRecords {
		if empty, err := r.setTypeFromFields(); err != nil || empty {
			return 0, err
		}
	}

	switch r.recordsType {
	case legacyRecords:
		if r.MsgSet == nil {
			return 0, nil
		}
		return len(r.MsgSet.Messages), nil
	case defaultRecords:
		if r.RecordBatch == nil {
			return 0, nil
		}
		return len(r.RecordBatch.Records), nil
	}
	return 0, fmt.Errorf("unknown records type: %v", r.recordsType)
}

func magicValue(pd PacketDecoder) (int8, error) {
	return pd.peekInt8(magicOffset)
}
