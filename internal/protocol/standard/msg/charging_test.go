package msg

import (
	"encoding/hex"
	"testing"
)

// TestChargerStopUpload_Decode_WithChargerPlaintext tests the 0x05 Decode
// using the actual plaintext data domain from the charger's log.
func TestChargerStopUpload_Decode_WithChargerPlaintext(t *testing.T) {
	// Charger's plaintext data domain (126 bytes, from charger log)
	// HEX报文 header + data: 320605D396B9050100407E00 + <data domain>
	// Data domain extracted from after the 12-byte header
	dataHex := "0000202604240156270100000000000000000069000000000000ECE00700004D000000000000000000330000000000000000000000000000000000000002202604241000E80300D007002A6201006A230000D546000004202604241022E80300D00700C27E060046A600008D4C0100042026042409562720260424102216"

	data, err := hex.DecodeString(dataHex)
	if err != nil {
		t.Fatalf("hex decode error: %v", err)
	}
	if len(data) != 126 {
		t.Fatalf("data length = %d, expected 126", len(data))
	}

	// Debug: print bytes at key offsets
	t.Logf("Data[62:93] (FeeModelItem0 + FeeModelItem1 EndTime): % X", data[62:93])
	t.Logf("Data[87:112] (FeeModelItem1 full): % X", data[87:112])
	t.Logf("Data[112:126] (ChargeStartTime + ChargeEndTime): % X", data[112:126])

	var msg ChargerStopUpload
	if err := msg.Decode(data); err != nil {
		t.Fatalf("Decode() error: %v", err)
	}

	// Verify key fields against charger's parsed JSON
	t.Run("chargeOrderNo", func(t *testing.T) {
		if msg.ChargeOrderNo != "00002026042401562701" {
			t.Errorf("got %q, expected %q", msg.ChargeOrderNo, "00002026042401562701")
		}
	})

	t.Run("deviceOrderNo", func(t *testing.T) {
		if msg.DeviceOrderNo != "00000000000000000069" {
			t.Errorf("got %q, expected %q", msg.DeviceOrderNo, "00000000000000000069")
		}
	})

	t.Run("type", func(t *testing.T) {
		if msg.Type != 0 {
			t.Errorf("got %d, expected 0", msg.Type)
		}
	})

	t.Run("startElectricMeter", func(t *testing.T) {
		if msg.StartElectricMeter != 0 {
			t.Errorf("got %d, expected 0", msg.StartElectricMeter)
		}
	})

	t.Run("stopElectricMeter", func(t *testing.T) {
		if msg.StopElectricMeter != 516332 {
			t.Errorf("got %d, expected 516332", msg.StopElectricMeter)
		}
	})

	t.Run("stopReason", func(t *testing.T) {
		if msg.StopReason != 77 {
			t.Errorf("got %d, expected 77", msg.StopReason)
		}
	})

	t.Run("stopSoc", func(t *testing.T) {
		if msg.StopSoc != 51 {
			t.Errorf("got %d, expected 51", msg.StopSoc)
		}
	})

	t.Run("messageCount", func(t *testing.T) {
		if msg.MessageCount != 2 {
			t.Errorf("got %d, expected 2", msg.MessageCount)
		}
	})

	t.Run("feeModelList_count", func(t *testing.T) {
		if len(msg.FeeModelList) != 2 {
			t.Fatalf("got %d items, expected 2", len(msg.FeeModelList))
		}
	})

	t.Run("feeModelItem0", func(t *testing.T) {
		fi := msg.FeeModelList[0]
		if fi.EndTime != "202604241000" {
			t.Errorf("endTime = %q, expected %q", fi.EndTime, "202604241000")
		}
		if fi.ElectricPrice != 1000 {
			t.Errorf("electricPrice = %d, expected 1000", fi.ElectricPrice)
		}
		if fi.ServicePrice != 2000 {
			t.Errorf("servicePrice = %d, expected 2000", fi.ServicePrice)
		}
		if fi.ElectricQuantity != 90666 {
			t.Errorf("electricQuantity = %d, expected 90666", fi.ElectricQuantity)
		}
		if fi.ElectricFee != 9066 {
			t.Errorf("electricFee = %d, expected 9066", fi.ElectricFee)
		}
		if fi.ServiceFee != 18133 {
			t.Errorf("serviceFee = %d, expected 18133", fi.ServiceFee)
		}
		if fi.ElectricFlag != 4 {
			t.Errorf("electricFlag = %d, expected 4", fi.ElectricFlag)
		}
	})

	t.Run("feeModelItem1", func(t *testing.T) {
		fi := msg.FeeModelList[1]
		if fi.EndTime != "202604241022" {
			t.Errorf("endTime = %q, expected %q", fi.EndTime, "202604241022")
		}
		if fi.ElectricPrice != 1000 {
			t.Errorf("electricPrice = %d, expected 1000", fi.ElectricPrice)
		}
		if fi.ServicePrice != 2000 {
			t.Errorf("servicePrice = %d, expected 2000", fi.ServicePrice)
		}
		if fi.ElectricQuantity != 425666 {
			t.Errorf("electricQuantity = %d, expected 425666", fi.ElectricQuantity)
		}
		if fi.ElectricFee != 42566 {
			t.Errorf("electricFee = %d, expected 42566", fi.ElectricFee)
		}
		if fi.ServiceFee != 85133 {
			t.Errorf("serviceFee = %d, expected 85133", fi.ServiceFee)
		}
		if fi.ElectricFlag != 4 {
			t.Errorf("electricFlag = %d, expected 4", fi.ElectricFlag)
		}
	})

	t.Run("chargeStartTime", func(t *testing.T) {
		if msg.ChargeStartTime != "20260424095627" {
			t.Errorf("got %q, expected %q", msg.ChargeStartTime, "20260424095627")
		}
	})

	t.Run("chargeEndTime", func(t *testing.T) {
		if msg.ChargeEndTime != "20260424102216" {
			t.Errorf("got %q, expected %q", msg.ChargeEndTime, "20260424102216")
		}
	})
}

// TestChargerStopUpload_Decode_WithSpecExample tests using the protocol spec example data.
func TestChargerStopUpload_Decode_WithSpecExample(t *testing.T) {
	// From the protocol spec (standard.MD lines 809-890)
	// Full frame HEX (header + data domain, encryptFlag=0):
	fullFrameHex := "320605EAB3320102006C6500002026021410152900010000000000177103533501962C3F17000A313F17004D0000000000040000003500008201284200000100FF00000000000000000120260214110073330096140074040000DD05000059020000012026021410155620260214101607"

	fullFrame, err := hex.DecodeString(fullFrameHex)
	if err != nil {
		t.Fatalf("hex decode error: %v", err)
	}

	// Extract data domain (after 12-byte header)
	data := fullFrame[12:]

	var msg ChargerStopUpload
	if err := msg.Decode(data); err != nil {
		t.Fatalf("Decode() error: %v", err)
	}

	// Verify against spec example JSON
	if msg.ChargeOrderNo != "00202602141015290001" {
		t.Errorf("chargeOrderNo = %q, expected %q", msg.ChargeOrderNo, "00202602141015290001")
	}
	if msg.DeviceOrderNo != "00000000001771035335" {
		t.Errorf("deviceOrderNo = %q, expected %q", msg.DeviceOrderNo, "00000000001771035335")
	}
	if msg.Type != 1 {
		t.Errorf("type = %d, expected 1", msg.Type)
	}
	if msg.StopReason != 77 {
		t.Errorf("stopReason = %d, expected 77", msg.StopReason)
	}
	if msg.StopSoc != 53 {
		t.Errorf("stopSoc = %d, expected 53", msg.StopSoc)
	}
	if msg.MessageCount != 1 {
		t.Errorf("messageCount = %d, expected 1", msg.MessageCount)
	}
	if len(msg.FeeModelList) != 1 {
		t.Fatalf("feeModelList count = %d, expected 1", len(msg.FeeModelList))
	}
	fi := msg.FeeModelList[0]
	if fi.EndTime != "202602141100" {
		t.Errorf("endTime = %q, expected %q", fi.EndTime, "202602141100")
	}
	if fi.ElectricPrice != 13171 {
		t.Errorf("electricPrice = %d, expected 13171", fi.ElectricPrice)
	}
	if fi.ServicePrice != 5270 {
		t.Errorf("servicePrice = %d, expected 5270", fi.ServicePrice)
	}
	if fi.ElectricQuantity != 1140 {
		t.Errorf("electricQuantity = %d, expected 1140", fi.ElectricQuantity)
	}
	if fi.ElectricFee != 1501 {
		t.Errorf("electricFee = %d, expected 1501", fi.ElectricFee)
	}
	if fi.ServiceFee != 601 {
		t.Errorf("serviceFee = %d, expected 601", fi.ServiceFee)
	}
	if fi.ElectricFlag != 1 {
		t.Errorf("electricFlag = %d, expected 1", fi.ElectricFlag)
	}
	if msg.ChargeStartTime != "20260214101556" {
		t.Errorf("chargeStartTime = %q, expected %q", msg.ChargeStartTime, "20260214101556")
	}
	if msg.ChargeEndTime != "20260214101607" {
		t.Errorf("chargeEndTime = %q, expected %q", msg.ChargeEndTime, "20260214101607")
	}
}
