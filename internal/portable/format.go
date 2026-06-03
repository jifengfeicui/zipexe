package portable

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
)

var Magic = [8]byte{'G', 'O', 'P', 'O', 'R', 'T', '0', '1'}

const FooterSize = 24

type Metadata struct {
	AppName      string `json:"app_name"`
	Entry        string `json:"entry"`
	CreatedAt    int64  `json:"created_at"`
	RequireAdmin bool   `json:"require_admin"`
}

type Footer struct {
	MetaSize    uint64
	PayloadSize uint64
}

func EncodeMetadata(meta Metadata) ([]byte, error) {
	return json.Marshal(meta)
}

func DecodeMetadata(data []byte) (Metadata, error) {
	var meta Metadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return Metadata{}, err
	}
	if meta.AppName == "" {
		return Metadata{}, errors.New("metadata app_name is empty")
	}
	if meta.Entry == "" {
		return Metadata{}, errors.New("metadata entry is empty")
	}
	return meta, nil
}

func WriteFooter(w io.Writer, footer Footer) error {
	buf := make([]byte, FooterSize)
	copy(buf[:8], Magic[:])
	binary.LittleEndian.PutUint64(buf[8:16], footer.MetaSize)
	binary.LittleEndian.PutUint64(buf[16:24], footer.PayloadSize)
	_, err := w.Write(buf)
	return err
}

func ReadFooter(r io.ReaderAt, size int64) (Footer, error) {
	if size < FooterSize {
		return Footer{}, errors.New("file is too small")
	}
	buf := make([]byte, FooterSize)
	if _, err := r.ReadAt(buf, size-FooterSize); err != nil {
		return Footer{}, err
	}
	var magic [8]byte
	copy(magic[:], buf[:8])
	if magic != Magic {
		return Footer{}, errors.New("portable payload not found")
	}
	return Footer{
		MetaSize:    binary.LittleEndian.Uint64(buf[8:16]),
		PayloadSize: binary.LittleEndian.Uint64(buf[16:24]),
	}, nil
}
