package hds

import (
	"encoding/binary"

	"github.com/brutella/hc/crypto/chacha20poly1305"
	"github.com/brutella/hc/crypto/hkdf"
)

type HDSSession struct {
	encryptKey [32]byte
	decryptKey [32]byte

	encryptCount uint64
	decryptCount uint64
}

func NewHDSSession(controllerKeySalt []byte, accessoryKeySalt []byte, sharedKey [32]byte) (HDSSession, error) {
	var salt []byte
	salt = append(salt, controllerKeySalt...)
	salt = append(salt, accessoryKeySalt...)

	var accessoryToControllerInfo = []byte("HDS-Read-Encryption-Key")
	var controllerToAccessoryInfo = []byte("HDS-Write-Encryption-Key")

	var sess HDSSession

	encryptKey, err := hkdf.Sha512(sharedKey[:], salt, accessoryToControllerInfo)
	if err != nil {
		return sess, err
	}

	sess.encryptKey = encryptKey
	sess.encryptCount = 0

	decryptKey, err := hkdf.Sha512(sharedKey[:], salt, controllerToAccessoryInfo)
	if err != nil {
		return sess, err
	}

	sess.decryptKey = decryptKey
	sess.decryptCount = 0

	return sess, nil
}

func (h *HDSSession) Encrypt(payload []byte, aad []byte) ([]byte, [16]byte, error) {
	var nonce [8]byte
	binary.LittleEndian.PutUint64(nonce[:], h.encryptCount)

	h.encryptCount++

	return chacha20poly1305.EncryptAndSeal(h.encryptKey[:], nonce[:], payload, aad)
}

func (h *HDSSession) Decrypt(payload []byte, mac [16]byte, aad []byte) ([]byte, error) {
	var nonce [8]byte
	binary.LittleEndian.PutUint64(nonce[:], h.decryptCount)

	h.decryptCount++

	return chacha20poly1305.DecryptAndVerify(h.decryptKey[:], nonce[:], payload, mac, aad)
}
