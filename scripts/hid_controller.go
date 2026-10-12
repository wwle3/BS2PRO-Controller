package main

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/sstallion/go-hid"
)

// 风扇数据结构
type FanData struct {
	ReportID             uint8  // 报告ID
	MagicSync            uint16 // 同步头 0x5AA5
	Command              uint8  // 命令码
	Status               uint8  // 状态字节
	GearSettings         uint8  // data[5]，vbus 在 bit5-6
	CurrentMode          uint8  // data[6]，智能变频、通电自启、智能启停
	Reserved1            uint8  // 预留字节
	CurrentRPM           uint16 // 风扇实时转速
	TargetRPM            uint16 // 风扇目标转速
	Vbus                 uint8
	SmartFreq            bool
	PowerOnAutoStart     bool
	IntelligentStartStop uint8
}

// 解析HID数据包
func parseFanData(data []byte, length int) *FanData {
	if length < 11 {
		fmt.Printf("数据包长度不足，需要至少11字节，实际: %d\n", length)
		return nil
	}

	// 检查同步头
	magic := binary.BigEndian.Uint16(data[1:3])
	if magic != 0x5AA5 {
		fmt.Printf("同步头不匹配，期望: 0x5AA5, 实际: 0x%04X\n", magic)
		return nil
	}

	statusByte := data[5]
	flagByte := data[6]
	fanData := &FanData{
		ReportID:             data[0],
		MagicSync:            magic,
		Command:              data[3],
		Status:               data[4],
		GearSettings:         statusByte,
		CurrentMode:          flagByte,
		Reserved1:            data[7],
		Vbus:                 (statusByte >> 5) & 3,
		SmartFreq:            flagByte&0x01 != 0,
		PowerOnAutoStart:     flagByte&0x02 != 0,
		IntelligentStartStop: (flagByte >> 2) & 3,
	}

	// 解析转速 (小端序)
	if length >= 10 {
		fanData.CurrentRPM = binary.LittleEndian.Uint16(data[8:10])
	}
	if length >= 12 {
		fanData.TargetRPM = binary.LittleEndian.Uint16(data[10:12])
	}

	return fanData
}

// 显示风扇数据
func displayFanData(fanData *FanData) {
	fmt.Println("\n=== 风扇数据解析 ===")
	fmt.Printf("报告ID: 0x%02X\n", fanData.ReportID)
	fmt.Printf("同步头: 0x%04X\n", fanData.MagicSync)
	fmt.Printf("命令码: 0x%02X\n", fanData.Command)
	fmt.Printf("状态字节: 0x%02X\n", fanData.Status)

	fmt.Printf("状态字节 data[5]: 0x%02X，VBUS: %d\n", fanData.GearSettings, fanData.Vbus)
	fmt.Printf("标志字节 data[6]: 0x%02X，智能变频: %t，通电自启: %t，智能启停: %d\n",
		fanData.CurrentMode, fanData.SmartFreq, fanData.PowerOnAutoStart, fanData.IntelligentStartStop)
	fmt.Printf("预留字节: 0x%02X\n", fanData.Reserved1)
	fmt.Printf("风扇实时转速: %d RPM\n", fanData.CurrentRPM)
	fmt.Printf("风扇目标转速: %d RPM\n", fanData.TargetRPM)
	fmt.Println("==================")
}

func main() {
	fmt.Println("HID连接测试")

	// 初始化HID库
	err := hid.Init()
	if err != nil {
		fmt.Printf("初始化HID库失败: %v\n", err)
		return
	}
	defer func() {
		// 清理HID库资源
		if err := hid.Exit(); err != nil {
			fmt.Printf("清理HID库失败: %v\n", err)
		}
	}()

	// 目标设备的厂商ID和产品ID
	vendorID := uint16(0x37D7)  // 厂商ID: 0x37D7 (corrected from 0x137D7)
	productID := uint16(0x1002) // 产品ID: 0x1002

	fmt.Printf("正在连接设备 - 厂商ID: 0x%04X, 产品ID: 0x%04X\n", vendorID, productID)

	// 直接打开第一个匹配的设备，无需枚举
	device, err := hid.OpenFirst(vendorID, productID)
	if err != nil {
		fmt.Printf("打开设备失败: %v\n", err)
		return
	}
	defer func() {
		if err := device.Close(); err != nil {
			fmt.Printf("关闭设备失败: %v\n", err)
		}
	}()

	fmt.Println("设备连接成功！")

	// 获取设备信息
	deviceInfo, err := device.GetDeviceInfo()
	if err != nil {
		fmt.Printf("获取设备信息失败: %v\n", err)
	} else {
		fmt.Printf("设备详细信息:\n")
		fmt.Printf("  制造商字符串: %s\n", deviceInfo.MfrStr)
		fmt.Printf("  产品字符串: %s\n", deviceInfo.ProductStr)
		fmt.Printf("  序列号: %s\n", deviceInfo.SerialNbr)
		fmt.Printf("  版本号: 0x%04X\n", deviceInfo.ReleaseNbr)
	}

	// 尝试读取数据（非阻塞模式）
	err = device.SetNonblock(true)
	if err != nil {
		fmt.Printf("设置非阻塞模式失败: %v\n", err)
	}

	// 读取示例
	buffer := make([]byte, 64)
	fmt.Println("尝试读取数据（超时5秒）...")

	n, err := device.ReadWithTimeout(buffer, 5*time.Second)
	if err != nil {
		if err == hid.ErrTimeout {
			fmt.Println("读取超时，设备可能没有发送数据")
		} else {
			fmt.Printf("读取数据失败: %v\n", err)
		}
	} else {
		fmt.Printf("读取到 %d 字节数据: ", n)
		for i := range n {
			fmt.Printf("%02X ", buffer[i])
		}
		fmt.Println()

		// 解析风扇数据
		if fanData := parseFanData(buffer, n); fanData != nil {
			displayFanData(fanData)
		}
	}

	// 发送数据示例（如果需要）
	// 第一个字节通常是报告ID，对于只支持单一报告的设备应为0
	// outputData := []byte{0x00, 0x01, 0x02, 0x03} // 示例数据
	// n, err = device.Write(outputData)
	// if err != nil {
	//     fmt.Printf("发送数据失败: %v\n", err)
	// } else {
	//     fmt.Printf("成功发送 %d 字节数据\n", n)
	// }

	fmt.Println("HID设备操作完成")
}
