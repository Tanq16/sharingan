package scaffold

type Action string

const (
	Created  Action = "created"
	Existing Action = "existing"
	Deleted  Action = "deleted"
)

type Event struct {
	Action   Action
	Resource string
	ID       string
}

type Config struct {
	Notify func(Event)
}

func (c Config) notify(action Action, resource, id string) {
	if c.Notify != nil {
		c.Notify(Event{Action: action, Resource: resource, ID: id})
	}
}

const (
	resVPC        = "vpc"
	resIGW        = "internet gateway"
	resSubnet     = "subnet"
	resRouteTable = "route table"
	resSecGroup   = "security group"
	resKeyPair    = "key pair"
)

// One scaffold per account and region, so a fixed name per resource collides with nothing.
const (
	nameVPC        = "sharingan-vpc"
	nameIGW        = "sharingan-igw"
	nameSubnet     = "sharingan-subnet"
	nameRouteTable = "sharingan-rtb"
	nameSecGroup   = "sharingan-sg"
	nameKeyPair    = "sharingan-key"
)
