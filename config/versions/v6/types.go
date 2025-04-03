package v6

// DefaultConfig is the default config used for the version 6 portfolio providers upgrade
var DefaultConfig = []byte(`[
	{
		"name": "Ethplorer",
		"enabled": true
	},
	{
		"name": "XRPScan",
		"enabled": true
	},
	{
		"name": "CryptoID",
		"enabled": false,
		"apiKey": "Key"
	}
]`)
