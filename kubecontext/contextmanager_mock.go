package kubecontext

type configContextManagerMock struct {
	currentContext string
	contextNames   []string
}

var _ ConfigContextManager = &configContextManagerMock{}

func CreateConfigContextManagerMock(names []string, current string) ConfigContextManager {
	return &configContextManagerMock{
		currentContext: current,
		contextNames:   names,
	}
}

func (c *configContextManagerMock) GetCurrentContext() string {
	return c.currentContext
}

func (c *configContextManagerMock) GetContextNames() ([]string, error) {
	return c.contextNames, nil
}

func (c *configContextManagerMock) SetCurrentContext(name string) error {
	c.currentContext = name
	return nil
}

func (c *configContextManagerMock) TearDown() error {
	return nil
}
