/*
*
* Copyright 2023-2025 mkrainbow.com.
*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
*
*      http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
*
 */

package deviceproto

import (
	"bytes"
	"reflect"
	"testing"

	"gotest.tools/assert"
)

func TestEncodeVerifyReq(t *testing.T) {
	// $ echo -n "cfa09baa-4913-4ad7-a936-2e26f9671b04:mb6bgso4EChvyzA05thF9+wH"| wc -L
	// 61
	// $ printf "%x\n" 61
	// 3d
	req := &VerifydReq{
		Header: &Header{
			Version: Version,
			Type:    MsgType_DeviceVerifyReq,
			ID:      0x8899,
			BodyLen: 0, // will update this field when decode
			Code:    4,
		},
		CapLevel:     1,
		DeviceID:     "cfa09baa-4913-4ad7-a936-2e26f9671b04",
		DeviceSecret: "mb6bgso4EChvyzA05thF9+wH",
	}

	buf, err := EncodeVerifyReq(req)
	assert.NilError(t, err)
	assert.Equal(t, len(buf), int(HeaderLen+1+0x3d))

	want := make([]byte, HeaderLen+1+0x3d)
	token := "cfa09baa-4913-4ad7-a936-2e26f9671b04" + ":" + "mb6bgso4EChvyzA05thF9+wH"
	copy(want[0:HeaderLen+1], []byte{0x14, 0x88, 0x99, 0x00, 0x3d + 1, 0x40})
	copy(want[HeaderLen+1:int(HeaderLen)+1+len(token)], token)
	if !bytes.Equal(buf, want) {
		t.Errorf("Encode() = %q, want %q", buf, want)
	}
	t.Logf("len=%d buf=%x", len(buf), buf)
}

func TestDecodeVerifyReq(t *testing.T) {

	token := "cfa09baa-4913-4ad7-a936-2e26f9671b04" + ":" + "mb6bgso4EChvyzA05thF9+wH"
	bodyLen := uint16(len(token)) + 1
	buf := make([]byte, HeaderLen+uint16(bodyLen))
	copy(buf[0:HeaderLen+1], []byte{0x14, 0x88, 0x99, 0x00, 0x3d + 1, 0x40})
	copy(buf[HeaderLen+1:int(HeaderLen)+1+len(token)], token)

	headerWant := &Header{
		Version: Version,
		Type:    MsgType_DeviceVerifyReq,
		ID:      0x8899,
		BodyLen: bodyLen,
		Code:    4,
	}
	reqWant := &VerifydReq{
		Header:       headerWant,
		CapLevel:     1,
		DeviceID:     "cfa09baa-4913-4ad7-a936-2e26f9671b04",
		DeviceSecret: "mb6bgso4EChvyzA05thF9+wH",
	}
	req, err := DecodeVerifyReq(buf)

	if err != nil {
		t.Error("DecodeVerifyReq err", err)
	}

	if !reflect.DeepEqual(req, reqWant) {
		t.Error("(req != reqWant):", req, reqWant)
	}
	t.Logf("req=%v", req)

}
func TestDecodeVerifyReqDeviceID(t *testing.T) {
	token := "cfa09baa-4913-4ad7-a936-2e26f9671b04f" + ":" + "mb6bgso4EChvyzA05thF9+wH"
	bodyLen := uint16(len(token)) + 1
	buf := make([]byte, HeaderLen+uint16(bodyLen))
	copy(buf[0:HeaderLen+1], []byte{0x14, 0x88, 0x99, 0x00, 0x3e + 1, 0x40})
	copy(buf[HeaderLen+1:int(HeaderLen)+1+len(token)], token)
	req, err := DecodeVerifyReq(buf)
	assert.Equal(t, err, ErrVerifyData)
	t.Logf("req=%v", req)
}
func TestDecodeVerifyReqDeviceSecret(t *testing.T) {
	{
		token := "cfa09baa-4913-4ad7-a936-2e26f9671b04" + ":" + "mb6bgso4EChvyzA05thF9+w"
		bodyLen := uint16(len(token)) + 1
		buf := make([]byte, HeaderLen+uint16(bodyLen))
		copy(buf[0:HeaderLen+1], []byte{0x14, 0x88, 0x99, 0x00, 0x3c + 1, 0x40})
		copy(buf[HeaderLen+1:int(HeaderLen)+1+len(token)], token)
		req, err := DecodeVerifyReq(buf)
		assert.Equal(t, err, ErrVerifyData)
		t.Logf("req=%v", req)
	}
	{
		token := "cfa09baa-4913-4ad7-a936-2e26f9671b04" + ":" + "12345678901234567890123456789012345678901234567890123456789012345"
		bodyLen := uint16(len(token)) + 1
		buf := make([]byte, HeaderLen+uint16(bodyLen))
		copy(buf[0:HeaderLen+1], []byte{0x14, 0x88, 0x99, 0x00, 0x3c + 1, 0x40})
		copy(buf[HeaderLen+1:int(HeaderLen)+1+len(token)], token)
		req, err := DecodeVerifyReq(buf)
		assert.Equal(t, err, ErrExceedLength)
		t.Logf("req=%v", req)
	}

}
func TestEncodeSendReq(t *testing.T) {
	body := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a}
	req := &SendReq{
		Header: &Header{
			Version: Version,
			Type:    MsgType_ServerSendReq,
			ID:      0x8899,
			BodyLen: 0, // will update this field when encode
			Code:    4,
		},
		Body: body,
	}

	buf, err := EncodeSendReq(req)
	assert.NilError(t, err)
	assert.Equal(t, len(buf), int(int(HeaderLen)+len(body)))
	want := make([]byte, int(HeaderLen)+len(body))
	copy(want[0:HeaderLen], []byte{0x74, 0x88, 0x99, 0x00, uint8(len(body))})
	copy(want[HeaderLen:int(HeaderLen)+len(body)], body)
	if !bytes.Equal(buf, want) {
		t.Errorf("Encode() = %q, want %q", buf, want)
	}
	t.Logf("len=%d buf=%x", len(buf), buf)
}
func TestDecodeSendReq(t *testing.T) {
	body := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a}
	buf := make([]byte, int(HeaderLen)+len(body))
	copy(buf[0:HeaderLen], []byte{0x74, 0x88, 0x99, 0x00, uint8(len(body))})
	copy(buf[HeaderLen:int(HeaderLen)+len(body)], body)

	req, err := DecodeSendReq(buf)
	assert.NilError(t, err)

	reqWant := &SendReq{
		Header: &Header{
			Version: Version,
			Type:    MsgType_ServerSendReq,
			ID:      0x8899,
			BodyLen: uint16(len(body)),
			Code:    4,
		},
		Body: body,
	}
	if !reflect.DeepEqual(req, reqWant) {
		t.Error("(req != reqWant):", req, reqWant)
	}
	t.Logf("req=%v", req)

}

func TestEncodeSendResp(t *testing.T) {
	body := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a}
	req := &SendResp{
		Header: &Header{
			Version: Version,
			Type:    MsgType_ServerSendResp,
			ID:      0x8899,
			BodyLen: 0, // will update this field when decode
			Code:    4,
		},
		Body: body,
	}

	buf, err := EncodeSendResp(req)
	assert.NilError(t, err)
	assert.Equal(t, len(buf), int(int(HeaderLen)+len(body)))
	want := make([]byte, int(HeaderLen)+len(body))
	copy(want[0:HeaderLen], []byte{0x84, 0x88, 0x99, 0x00, uint8(len(body))})
	copy(want[HeaderLen:int(HeaderLen)+len(body)], body)
	if !bytes.Equal(buf, want) {
		t.Errorf("Encode() = %q, want %q", buf, want)
	}
	t.Logf("len=%d buf=%x", len(buf), buf)
}
func TestDecodeSendResp(t *testing.T) {
	body := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a}
	buf := make([]byte, int(HeaderLen)+len(body))
	copy(buf[0:HeaderLen], []byte{0x84, 0x88, 0x99, 0x00, uint8(len(body))})
	copy(buf[HeaderLen:int(HeaderLen)+len(body)], body)

	req, err := DecodeSendResp(buf)
	assert.NilError(t, err)

	reqWant := &SendResp{
		Header: &Header{
			Version: Version,
			Type:    MsgType_ServerSendResp,
			ID:      0x8899,
			BodyLen: uint16(len(body)),
			Code:    4,
		},
		Body: body,
	}
	if !reflect.DeepEqual(req, reqWant) {
		t.Error("(req != reqWant):", req, reqWant)
	}
	t.Logf("req=%v", req)

}
