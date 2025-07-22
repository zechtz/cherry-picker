package main

func (cp *CherryPicker) setup() error {
	if err := cp.validateBranch(); err != nil {
		return err
	}

	if err := cp.fetchOrigin(); err != nil {
		return err
	}

	if err := cp.getUniqueCommits(); err != nil {
		return err
	}

	return nil
}