package statsdproxy

import (
	"reflect"
	"testing"
)

func TestReadConfigFile_BasicData(t *testing.T) {
	cases := []struct {
		Input          string
		ExpectedOutput *ProxyConfig
	}{
		{
			`
			{
			"nodes": [
				{"host": "127.0.0.1", "port": 8126, "adminport": 8136},
				{"host": "127.0.0.1", "port": 8127, "adminport": 8137},
				{"host": "127.0.0.1", "port": 8128, "adminport": 8138}
			],
			"host":  "0.0.0.0",
			"port": 8125,
			"managementPort": 8135,
			"checkInterval": 1000,
			"cacheSize": 10000
			}
			`,
			&ProxyConfig{
				Nodes: []StatsdConfigNode{
					{
						Host:      "127.0.0.1",
						Port:      8126,
						AdminPort: 8136,
					},
					{
						Host:      "127.0.0.1",
						Port:      8127,
						AdminPort: 8137,
					},
					{
						Host:      "127.0.0.1",
						Port:      8128,
						AdminPort: 8138,
					},
				},
				Host:           "0.0.0.0",
				Port:           8125,
				ManagementPort: 8135,
				CheckInterval:  1000,
			},
		},
		{
			`
			{
			"nodes": [
				{"host": "127.0.0.1", "port": 8126},
				{"host": "127.0.0.1", "port": 8127},
				{"host": "127.0.0.1", "port": 8128, "adminport": 8138}
			],
			"host":  "0.0.0.0",
			"port": 8125,
			"managementPort": 8135,
			"checkInterval": 1000,
			"cacheSize": 10000
			}
			`,
			&ProxyConfig{
				Nodes: []StatsdConfigNode{
					{
						Host:      "127.0.0.1",
						Port:      8126,
						AdminPort: 0,
					},
					{
						Host:      "127.0.0.1",
						Port:      8127,
						AdminPort: 0,
					},
					{
						Host:      "127.0.0.1",
						Port:      8128,
						AdminPort: 8138,
					},
				},
				Host:           "0.0.0.0",
				Port:           8125,
				ManagementPort: 8135,
				CheckInterval:  1000,
			},
		}}

	for _, tc := range cases {
		parsedConfig, err := readConfigFile([]byte(tc.Input))
		if err != nil {
			t.Errorf("readConfigFile() parsing is broken with %v", err)
			t.FailNow()
		}
		if !reflect.DeepEqual(parsedConfig, tc.ExpectedOutput) {
			t.Fatalf("Unexpected output from config reader.\nExpected: %#v\nGiven:    %#v",
				tc.ExpectedOutput, parsedConfig)
		}
	}
}

// Benchmarks
func BenchmarkReadConfigFile(b *testing.B) {
	const testConfig = `
    {
      "nodes": [
        {"host": "127.0.0.1", "port": 8129, "adminport": 8126},
        {"host": "127.0.0.1", "port": 8127, "adminport": 8128},
        {"host": "127.0.0.1", "port": 8129, "adminport": 8130}
      ],
      "host":  "0.0.0.0",
      "port": 8125,
      "checkInterval": 1000,
      "cacheSize": 10000
    }
    `

	for i := 0; i < b.N; i++ {
		readConfigFile([]byte(testConfig))
	}
}
