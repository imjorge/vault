package vault

import (
	"encoding/base64"
	"testing"

	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/vault/helper/pgpkeys"
	"github.com/hashicorp/vault/helper/xor"
)

func TestCore_RootGeneration_Lifecycle(t *testing.T) {
	c, master, _ := TestCoreUnsealed(t)

	// Verify update not allowed
	if _, err := c.RootGenerationUpdate(master, ""); err == nil {
		t.Fatalf("no root generation in progress")
	}

	// Should be no progress
	num, err := c.RootGenerationProgress()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if num != 0 {
		t.Fatalf("bad: %d", num)
	}

	// Should be no config
	conf, err := c.RootGenerationConfiguration()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if conf != nil {
		t.Fatalf("bad: %v", conf)
	}

	// Cancel should be idempotent
	err = c.RootGenerationCancel()
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	otpBytes, err := xor.GenerateRandBytes(16)
	if err != nil {
		t.Fatal(err)
	}

	// Start a root generation
	err = c.RootGenerationInit(base64.StdEncoding.EncodeToString(otpBytes), "")
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Should get config
	conf, err = c.RootGenerationConfiguration()
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Cancel should be clear
	err = c.RootGenerationCancel()
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Should be no config
	conf, err = c.RootGenerationConfiguration()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if conf != nil {
		t.Fatalf("bad: %v", conf)
	}
}

func TestCore_RootGeneration_Init(t *testing.T) {
	c, _, _ := TestCoreUnsealed(t)

	otpBytes, err := xor.GenerateRandBytes(16)
	if err != nil {
		t.Fatal(err)
	}

	err = c.RootGenerationInit(base64.StdEncoding.EncodeToString(otpBytes), "")
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Second should fail
	err = c.RootGenerationInit("", pgpkeys.TestPubKey1)
	if err == nil {
		t.Fatalf("should fail")
	}
}

func TestCore_RootGeneration_InvalidMaster(t *testing.T) {
	c, master, _ := TestCoreUnsealed(t)

	otpBytes, err := xor.GenerateRandBytes(16)
	if err != nil {
		t.Fatal(err)
	}

	err = c.RootGenerationInit(base64.StdEncoding.EncodeToString(otpBytes), "")
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Fetch new config with generated nonce
	rgconf, err := c.RootGenerationConfiguration()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if rgconf == nil {
		t.Fatalf("bad: no rekey config received")
	}

	// Provide the master (invalid)
	master[0]++
	_, err = c.RootGenerationUpdate(master, rgconf.Nonce)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestCore_RootGeneration_InvalidNonce(t *testing.T) {
	c, master, _ := TestCoreUnsealed(t)

	otpBytes, err := xor.GenerateRandBytes(16)
	if err != nil {
		t.Fatal(err)
	}

	err = c.RootGenerationInit(base64.StdEncoding.EncodeToString(otpBytes), "")
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Provide the nonce (invalid)
	_, err = c.RootGenerationUpdate(master, "abcd")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestCore_RootGeneration_Update_OTP(t *testing.T) {
	c, master, _ := TestCoreUnsealed(t)

	otpBytes, err := xor.GenerateRandBytes(16)
	if err != nil {
		t.Fatal(err)
	}

	otp := base64.StdEncoding.EncodeToString(otpBytes)
	// Start a root generation
	err = c.RootGenerationInit(otp, "")
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Fetch new config with generated nonce
	rkconf, err := c.RootGenerationConfiguration()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if rkconf == nil {
		t.Fatalf("bad: no root generation config received")
	}

	// Provide the master
	result, err := c.RootGenerationUpdate(master, rkconf.Nonce)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if result == nil {
		t.Fatalf("Bad, result is nil")
	}

	encodedRootToken := result.EncodedRootToken

	// Should be no progress
	num, err := c.RootGenerationProgress()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if num != 0 {
		t.Fatalf("bad: %d", num)
	}

	// Should be no config
	conf, err := c.RootGenerationConfiguration()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if conf != nil {
		t.Fatalf("bad: %v", conf)
	}

	tokenBytes, err := xor.XORBase64(encodedRootToken, otp)
	if err != nil {
		t.Fatal(err)
	}
	token, err := uuid.FormatUUID(tokenBytes)
	if err != nil {
		t.Fatal(err)
	}

	// Ensure that the token is a root token
	te, err := c.tokenStore.Lookup(token)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if te == nil {
		t.Fatalf("token was nil")
	}
	if te.ID != token || te.Parent != "" ||
		len(te.Policies) != 1 || te.Policies[0] != "root" {
		t.Fatalf("bad: %#v", *te)
	}
}

func TestCore_RootGeneration_Update_PGP(t *testing.T) {
	c, master, _ := TestCoreUnsealed(t)

	// Start a root generation
	err := c.RootGenerationInit("", pgpkeys.TestPubKey1)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Fetch new config with generated nonce
	rkconf, err := c.RootGenerationConfiguration()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if rkconf == nil {
		t.Fatalf("bad: no root generation config received")
	}

	// Provide the master
	result, err := c.RootGenerationUpdate(master, rkconf.Nonce)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if result == nil {
		t.Fatalf("Bad, result is nil")
	}

	encodedRootToken := result.EncodedRootToken

	// Should be no progress
	num, err := c.RootGenerationProgress()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if num != 0 {
		t.Fatalf("bad: %d", num)
	}

	// Should be no config
	conf, err := c.RootGenerationConfiguration()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if conf != nil {
		t.Fatalf("bad: %v", conf)
	}

	ptBuf, err := pgpkeys.DecryptBytes(encodedRootToken, pgpkeys.TestPrivKey1)
	if err != nil {
		t.Fatal(err)
	}
	if ptBuf == nil {
		t.Fatal("Got nil plaintext key")
	}

	token := ptBuf.String()

	// Ensure that the token is a root token
	te, err := c.tokenStore.Lookup(token)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if te == nil {
		t.Fatalf("token was nil")
	}
	if te.ID != token || te.Parent != "" ||
		len(te.Policies) != 1 || te.Policies[0] != "root" {
		t.Fatalf("bad: %#v", *te)
	}
}
