package topo

// The mapreduce topo splits into three layer 
// Layer 1 : 
// Mapper Layer 
// Layer 2 : 
// Shuffle Layer depends on all Mapper Nodes
// Layer 3 :
// Reducer Layer
// Shuffle Layer divide fairly to every Reducer node
// Prefix and Suffix array represents the dependency relationship between layers

type MapReduceTopology struct {
	NumOfMapper uint64
	NumOfReducer uint64
	NumOfShuffle uint64
	taskID     uint64
	prefix, suffix  []uint64
}

func (t *MapReduceTopology) SetTaskID(taskID uint64) {
	t.taskID = taskID
	var numOfPrefix uint64
	var numOfSuffix uint64
	var scopeL uint64
	shardQuotient, shardReminder := t.NumOfShuffle / t.NumOfReducer, t.NumOfShuflle % t.NumOfReducer
	switch {
		case taskID < t.NumOfMapper : 
			numOfPrefix = 0
			scopeL = 0
		case taskID < t.NumOfMapper + t.NumOfShuffle :
			numOfPrefix = t.NumOfMapper
			scopeL = 0
			scopeR = t.NumOfMapper 
		case taskID < t.NumOfMapper + t.NumOfShuffle + shardReminder:
			numOfPrefix = shardQuotient + 1
			scopeL = t.NumOfMapper + (shardQuotient + 1) * (taskID - t.NumOfMapper - t.NumOfShuffle)
		case tasdID < t.NumOfMapper + t.NumOfShuffle + t.NumberOfReduer :	
			numOfPrefix = t.NumOfShuffle / t.NumOfReducer
			scopeL = t.NumOfMapper + t.NumOfShuffle % t.NumOfReducer * (shardQuotient + 1)
			scopeL += (taskID - t.NumOfMapper - t.NumOfShuffle - shardReminder) * shardQoutient
		// default :
		// 	numOfPrefix = t.NumOfReducer 
		// 	scopeL = t.NumberOfMapper + t.NumberOfShuffle
	}
	t.prefix = make([]uint64, 0, numOfPrefix)
	for index := scopeL; index < scopeL + numOfPrefix; index++ {
		t.prefix = append(t.prefix, index)
	}

	switch {
		case taskID < t.NumOfMapper :
			numOfSuffix = t.NumOfShuffle
			scopeL = t.NumOfMapper 
		case taskID < t.NumOfMapper + t.NumOfShuffle :
			numOfsuffix = 1
			tmpAcc := taskID - t.NumberOfMapper 
			if tmpAcc / (shardQuotient + 1) < shardReminder {
				scopeL = tmpAcc / (shardQuotient + 1) + t.NumOfMapper
			} else {
				scopeL = tmpAcc - shardReminder * (shardQuotient + 1)
				scopeL += scopeL / shardQuotient + t.NumberOfMapper + shardReminder	
			} 
		case taskID < t.NumOfMapper + t.NumOfShuffle + t.NumOfReducer :
			numOfSuffix = 0
	}

	t.suffix = make([]uint64, 0, numOfSuffix)
	for index := scopeL; index < scopeL + numOfSuffix; index++ {
		t.suffix = append(t.suffix, index)
	}
}

func (t *MapReduceTopology) GetLinkTypes() []string {
	return []string{"Prefix", "Suffix"}
	// return []string{"Prefix"}
}

func (t *MapReduceTopology) GetNeighbors(linkType string, epoch uint64) []uint64 {
	res := make([]uint64, 0)
	switch {
	case linkType == "Prefix":
		res = t.prefix
	case linkType == "Suffix":
		res = t.suffix 
	}
	return res
}

// TODO, do we really need to expose this?

func (t *MapReduceTopology) SetNumberOfMapper(n uint64) { t.NumOfMapper = n }
func (t *MapReduceTopology) SetNumberOfShuffle(n uint64) { t.NumOfShuffle = n }
func (t *MapReduceTopology) SetNumberOfReduer(n uint64) { t.NumOfReducer = n }


// Creates a new tree topology with given fanout and number of tasks.
// This will be called during the task graph configuration.
func NewMaperTopology(nm, ns, nr uint64) *FullTopology {
	m := &MapReduceTopology{
		NumOfMapper: nm
		NumOfShuffle : ns
		NumOfReducer : nr
	}
	return m
}

