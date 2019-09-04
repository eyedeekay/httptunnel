package i2pbrowserproxy

import (
	"fmt"
	//	"log"
	"strconv"
	"strings"
)

//Option is a client Option
type Option func(*SAMMultiProxy) error

//SetName sets a clients's address in the form host:port or host, port
func SetName(s string) func(*SAMMultiProxy) error {
	return func(c *SAMMultiProxy) error {
		c.Conf.TunName = s
		return nil
	}
}

//SetAddr sets a clients's address in the form host:port or host, port
func SetAddr(s ...string) func(*SAMMultiProxy) error {
	return func(c *SAMMultiProxy) error {
		if len(s) == 1 {
			split := strings.SplitN(s[0], ":", 2)
			if len(split) == 2 {
				if i, err := strconv.Atoi(split[1]); err == nil {
					if i < 65536 {
						c.Conf.SamHost = split[0]
						c.Conf.SamPort = split[1]
						return nil
					}
					return fmt.Errorf("Invalid port")
				}
				return fmt.Errorf("Invalid port; non-number")
			}
			return fmt.Errorf("Invalid address; use host:port %s ", split)
		} else if len(s) == 2 {
			if i, err := strconv.Atoi(s[1]); err == nil {
				if i < 65536 {
					c.Conf.SamHost = s[0]
					c.Conf.SamPort = s[1]
					return nil
				}
				return fmt.Errorf("Invalid port")
			}
			return fmt.Errorf("Invalid port; non-number")
		} else {
			return fmt.Errorf("Invalid address")
		}
	}
}

//SetControlAddr sets a clients's address in the form host:port or host, port
func SetControlAddr(s ...string) func(*SAMMultiProxy) error {
	return func(c *SAMMultiProxy) error {
		if len(s) == 1 {
			split := strings.SplitN(s[0], ":", 2)
			if len(split) == 2 {
				if i, err := strconv.Atoi(split[1]); err == nil {
					if i < 65536 {
						c.Conf.ControlHost = split[0]
						c.Conf.ControlPort = split[1]
						return nil
					}
					return fmt.Errorf("Invalid port")
				}
				return fmt.Errorf("Invalid port; non-number")
			}
			return fmt.Errorf("Invalid address; use host:port %s ", split)
		} else if len(s) == 2 {
			if i, err := strconv.Atoi(s[1]); err == nil {
				if i < 65536 {
					c.Conf.ControlHost = s[0]
					c.Conf.ControlPort = s[1]
					return nil
				}
				return fmt.Errorf("Invalid port")
			}
			return fmt.Errorf("Invalid port; non-number")
		} else {
			return fmt.Errorf("Invalid address")
		}
	}
}

//SetProxyAddr sets a clients's address in the form host:port or host, port
func SetProxyAddr(s ...string) func(*SAMMultiProxy) error {
	return func(c *SAMMultiProxy) error {
		if len(s) == 1 {
			split := strings.SplitN(s[0], ":", 2)
			if len(split) == 2 {
				if i, err := strconv.Atoi(split[1]); err == nil {
					if i < 65536 {
						c.Conf.TargetHost = split[0]
						c.Conf.TargetPort = split[1]
						return nil
					}
					return fmt.Errorf("Invalid port")
				}
				return fmt.Errorf("Invalid port; non-number")
			}
			return fmt.Errorf("Invalid address; use host:port %s ", split)
		} else if len(s) == 2 {
			if i, err := strconv.Atoi(s[1]); err == nil {
				if i < 65536 {
					c.Conf.TargetHost = s[0]
					c.Conf.TargetPort = s[1]
					return nil
				}
				return fmt.Errorf("Invalid port")
			}
			return fmt.Errorf("Invalid port; non-number")
		} else {
			return fmt.Errorf("Invalid address")
		}
	}
}

//SetAddrMixed sets a clients's address in the form host, port(int)
func SetAddrMixed(s string, i int) func(*SAMMultiProxy) error {
	return func(c *SAMMultiProxy) error {
		if i < 65536 && i > 0 {
			c.Conf.SamHost = s
			c.Conf.SamPort = strconv.Itoa(i)
			return nil
		}
		return fmt.Errorf("Invalid port")
	}
}

//SetContrlHost sets the host of the client's Proxy Controller
func SetControlHost(s string) func(*SAMMultiProxy) error {
	return func(c *SAMMultiProxy) error {
		c.Conf.ControlHost = s
		return nil
	}
}

//SetKeysPath sets the path to the key save files
func SetKeysPath(s string) func(*SAMMultiProxy) error {
	return func(c *SAMMultiProxy) error {
		c.Conf.KeyFilePath = s
		return nil
	}
}

//SetContrlPort sets the host of the client's Proxy Controller
func SetControlPort(s string) func(*SAMMultiProxy) error {
	return func(c *SAMMultiProxy) error {
		port, err := strconv.Atoi(s)
		if err != nil {
			return fmt.Errorf("Invalid port; non-number")
		}
		if port < 65536 && port > -1 {
			c.Conf.ControlPort = s
			return nil
		}
		return fmt.Errorf("Invalid port")
	}
}

//SetHost sets the host of the client's SAM bridge
func SetHost(s string) func(*SAMMultiProxy) error {
	return func(c *SAMMultiProxy) error {
		c.Conf.SamHost = s
		return nil
	}
}

//SetPort sets the port of the client's SAM bridge using a string
func SetPort(s string) func(*SAMMultiProxy) error {
	return func(c *SAMMultiProxy) error {
		port, err := strconv.Atoi(s)
		if err != nil {
			return fmt.Errorf("Invalid port; non-number")
		}
		if port < 65536 && port > -1 {
			c.Conf.SamPort = s
			return nil
		}
		return fmt.Errorf("Invalid port")
	}
}

//SetPortInt sets the port of the client's SAM bridge using a string
func SetPortInt(i int) func(*SAMMultiProxy) error {
	return func(c *SAMMultiProxy) error {
		if i < 65536 && i > -1 {
			c.Conf.SamPort = strconv.Itoa(i)
			return nil
		}
		return fmt.Errorf("Invalid port")
	}
}

//SetDebug enables debugging messages
func SetDebug(b bool) func(*SAMMultiProxy) error {
	return func(c *SAMMultiProxy) error {
		c.debug = b
		return nil
	}
}

//SetInLength sets the number of hops inbound
func SetInLength(u uint) func(*SAMMultiProxy) error {
	return func(c *SAMMultiProxy) error {
		if u < 7 {
			c.Conf.InLength = int(u)
			return nil
		}
		return fmt.Errorf("Invalid inbound tunnel length")
	}
}

//SetOutLength sets the number of hops outbound
func SetOutLength(u uint) func(*SAMMultiProxy) error {
	return func(c *SAMMultiProxy) error {
		if u < 7 {
			c.Conf.OutLength = int(u)
			return nil
		}
		return fmt.Errorf("Invalid outbound tunnel length")
	}
}

//SetInVariance sets the variance of a number of hops inbound
func SetInVariance(i int) func(*SAMMultiProxy) error {
	return func(c *SAMMultiProxy) error {
		if i < 7 && i > -7 {
			c.Conf.InVariance = int(i)
			return nil
		}
		return fmt.Errorf("Invalid inbound tunnel length")
	}
}

//SetOutVariance sets the variance of a number of hops outbound
func SetOutVariance(i int) func(*SAMMultiProxy) error {
	return func(c *SAMMultiProxy) error {
		if i < 7 && i > -7 {
			c.Conf.OutVariance = int(i)
			return nil
		}
		return fmt.Errorf("Invalid outbound tunnel variance")
	}
}

//SetInQuantity sets the inbound tunnel quantity
func SetInQuantity(u uint) func(*SAMMultiProxy) error {
	return func(c *SAMMultiProxy) error {
		if u <= 16 {
			c.Conf.InQuantity = int(u)
			return nil
		}
		return fmt.Errorf("Invalid inbound tunnel quantity")
	}
}

//SetOutQuantity sets the outbound tunnel quantity
func SetOutQuantity(u uint) func(*SAMMultiProxy) error {
	return func(c *SAMMultiProxy) error {
		if u <= 16 {
			c.Conf.OutQuantity = int(u)
			return nil
		}
		return fmt.Errorf("Invalid outbound tunnel quantity")
	}
}

//SetInBackups sets the inbound tunnel backups
func SetInBackups(u uint) func(*SAMMultiProxy) error {
	return func(c *SAMMultiProxy) error {
		if u < 6 {
			c.Conf.InBackupQuantity = int(u)
			return nil
		}
		return fmt.Errorf("Invalid inbound tunnel backup quantity")
	}
}

//SetOutBackups sets the inbound tunnel backups
func SetOutBackups(u uint) func(*SAMMultiProxy) error {
	return func(c *SAMMultiProxy) error {
		if u < 6 {
			c.Conf.OutBackupQuantity = int(u)
			return nil
		}
		return fmt.Errorf("Invalid outbound tunnel backup quantity")
	}
}

//SetUnpublished tells the router to not publish the client leaseset
func SetUnpublished(b bool) func(*SAMMultiProxy) error {
	return func(c *SAMMultiProxy) error {
		c.Conf.Client = b
		return nil
	}
}

//SetEncrypt tells the router to use an encrypted leaseset
func SetEncrypt(b bool) func(*SAMMultiProxy) error {
	return func(c *SAMMultiProxy) error {
		c.Conf.EncryptLeaseSet = b
		return nil
	}
}

//SetReduceIdle sets the created tunnels to be reduced during extended idle time to avoid excessive resource usage
func SetReduceIdle(b bool) func(*SAMMultiProxy) error {
	return func(c *SAMMultiProxy) error {
		c.Conf.ReduceIdle = b
		return nil
	}
}

//SetReduceIdleTime sets time to wait before the tunnel quantity is reduced
func SetReduceIdleTime(u uint) func(*SAMMultiProxy) error {
	return func(c *SAMMultiProxy) error {
		if u > 299999 {
			c.Conf.ReduceIdleTime = int(u)
			return nil
		}
		return fmt.Errorf("Invalid reduce idle time %v", u)
	}
}

//SetReduceIdleQuantity sets number of tunnels to keep alive during an extended idle period
func SetReduceIdleQuantity(u uint) func(*SAMMultiProxy) error {
	return func(c *SAMMultiProxy) error {
		if u < 5 {
			c.Conf.ReduceIdleQuantity = int(u)
			return nil
		}
		return fmt.Errorf("Invalid reduced tunnel quantity %v", u)
	}
}

//SetCompression sets the tunnels to close after a specific amount of time
func SetCompression(b bool) func(*SAMMultiProxy) error {
	return func(c *SAMMultiProxy) error {
		c.Conf.UseCompression = b
		return nil
	}
}

//SetProxyMode tells whether to use per-ID or per-domain isolation
func SetProxyMode(b bool) func(*SAMMultiProxy) error {
	return func(c *SAMMultiProxy) error {
		c.aggressive = b
		return nil
	}
}

//return the inbound length as a string.
func (c *SAMMultiProxy) inlength() string {
	return fmt.Sprintf("inbound.length=%d", c.Conf.InLength)
}

//return the outbound length as a string.
func (c *SAMMultiProxy) outlength() string {
	return fmt.Sprintf("outbound.length=%d", c.Conf.OutLength)
}

//return the inbound length variance as a string.
func (c *SAMMultiProxy) invariance() string {
	return fmt.Sprintf("inbound.lengthVariance=%d", c.Conf.InVariance)
}

//return the outbound length variance as a string.
func (c *SAMMultiProxy) outvariance() string {
	return fmt.Sprintf("outbound.lengthVariance=%d", c.Conf.OutVariance)
}

//return the inbound tunnel quantity as a string.
func (c *SAMMultiProxy) inquantity() string {
	return fmt.Sprintf("inbound.quantity=%d", c.Conf.InQuantity)
}

//return the outbound tunnel quantity as a string.
func (c *SAMMultiProxy) outquantity() string {
	return fmt.Sprintf("outbound.quantity=%d", c.Conf.OutQuantity)
}

//return the inbound tunnel quantity as a string.
func (c *SAMMultiProxy) inbackups() string {
	return fmt.Sprintf("inbound.backupQuantity=%d", c.Conf.InQuantity)
}

//return the outbound tunnel quantity as a string.
func (c *SAMMultiProxy) outbackups() string {
	return fmt.Sprintf("outbound.backupQuantity=%d", c.Conf.OutQuantity)
}

func (c *SAMMultiProxy) encryptlease() string {
	if c.Conf.EncryptLeaseSet {
		return "i2cp.encryptLeaseSet=true"
	}
	return "i2cp.encryptLeaseSet=false"
}

func (c *SAMMultiProxy) dontpublishlease() string {
	if c.Conf.Client {
		return "i2cp.dontPublishLeaseSet=true"
	}
	return "i2cp.dontPublishLeaseSet=false"
}

func (c *SAMMultiProxy) reduceonidle() string {
	if c.Conf.ReduceIdle {
		return "i2cp.reduceOnIdle=true"
	}
	return "i2cp.reduceOnIdle=false"
}

func (c *SAMMultiProxy) reduceidletime() string {
	return fmt.Sprintf("i2cp.reduceIdleTime=%d", c.Conf.ReduceIdleTime)
}

func (c *SAMMultiProxy) reduceidlecount() string {
	return fmt.Sprintf("i2cp.reduceIdleQuantity=%d", c.Conf.ReduceIdleQuantity)
}

func (c *SAMMultiProxy) usecompresion() string {
	if c.Conf.UseCompression {
		return "i2cp.gzip=true"
	}
	return "i2cp.gzip=false"
}
