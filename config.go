package diffjson

var defaultConfig = Config{
	// TODO: GlobalIgnoreArrayOrder: false,
	GlobalIgnoreNumberType: true,
}

type Config struct {
	IgnorePath           []string
	IgnoreArrayOrderPath []string

	GlobalIgnoreArrayOrder bool
	GlobalIgnoreNumberType bool

	OmitEqual bool
}

func (c *Config) skip(path string) bool {
	// if contains(c.IgnorePath, strings.Trim(path, ".")) {
	if contains(c.IgnorePath, path) {
		return true
	}

	return false
}

func (c *Config) omit(res *Result) bool {
	if res == nil || (c.OmitEqual && res.Relation == EQUAL) {
		return true
	}
	return false
}

func contains(arr []string, target string) bool {
	for i := range arr {
		if arr[i] == target {
			return true
		}
	}
	return false
}
