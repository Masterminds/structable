package structable

/*
Hook func have the same signature:

func (this *Struct) () error

If hook call return a error, the caller stops execution and return the error.
For example

type Struct struct {
	Mandatory string `stbl:mandatory`
}

func (this *Struct) BeforeInserter() error {
	if this.Mandatory == "" {
		return errors.New("Mandatory field is empty!")
	}
	return nil
}
*/

// Called after Load() and LoadWhere()
type AfterLoader interface {
	AfterLoad() error
}

// Called before Insert()
type BeforeInserter interface {
	BeforeInsert() error
}

// Called after Insert()
type AfterInserter interface {
	AfterInsert() error
}

// Called before Update()
type BeforeUpdater interface {
	BeforeUpdate() error
}

// Called after Update()
type AfterUpdater interface {
	AfterUpdate() error
}

// Called before Delete()
type BeforeDeleter interface {
	BeforeDelete() error
}
