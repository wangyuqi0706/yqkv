package raft

import (
	"log"
	pb "raft/raftpb"
	"sort"
)

type raftLog struct {
	*pb.Log
}

func (l *raftLog) LastIncludedIndex() uint64 {
	return l.IncludedIndex
}

func (l *raftLog) LastIncludedTerm() uint64 {
	return l.IncludedTerm
}

func newLog(log *pb.Log) *raftLog {
	l := raftLog{log}
	return &l
}

func (l *raftLog) EntryAt(index uint64) pb.Entry {
	if index < 0 {
		log.Panicf("RAFT LOG: BAD LOG INDEX:[%v]", index)
	}
	if index == 0 {
		return pb.Entry{}
	}
	if index == l.IncludedIndex {
		return pb.Entry{
			Term:  l.IncludedTerm,
			Index: l.IncludedIndex,
			Data:  nil,
		}
	}
	if l.sliceIndex(index) < 0 {
		log.Panicf("RAFT LOG:INDEX [%v] HAS BEEN TRIMMED", index)
	}
	return *l.Entries[l.sliceIndex(index)]
}

func (l *raftLog) LastEntry() pb.Entry {
	if len(l.Entries) > 0 {
		return *l.Entries[len(l.Entries)-1]
	} else {
		return pb.Entry{Term: l.IncludedTerm, Index: l.IncludedIndex}
	}
}

func (l *raftLog) Append(entry pb.Entry) {
	if entry.Index == l.Len()+1 {
		l.Entries = append(l.Entries, &entry)
	}
}

func (l *raftLog) EntriesAfter(index uint64) []*pb.Entry {
	return l.Entries[l.sliceIndex(index):]
}

func (l *raftLog) CompactForSnapshot(index, term uint64) {
	if index > l.Len() {
		l.Entries = []*pb.Entry{}
	} else {
		l.Entries = l.Entries[l.sliceIndex(index)+1:]
	}
	l.IncludedIndex = index
	l.IncludedTerm = term
}

func (l *raftLog) DiscardAfterAnd(index uint64) {
	l.Entries = l.Entries[0:l.sliceIndex(index)]
}

func (l *raftLog) Len() uint64 {
	return l.LastEntry().Index
}

func (l *raftLog) FirstEntryAtTerm(term uint64) *pb.Entry {
	sliceIndex := sort.Search(len(l.Entries), func(i int) bool {
		return l.Entries[i].Term >= term
	})

	var entry *pb.Entry
	if sliceIndex < len(l.Entries) && l.Entries[sliceIndex].Term == term {
		entry = l.Entries[sliceIndex]
	} else {
		return nil
	}
	return entry
}

func (l *raftLog) LastEntryAtTerm(term uint64) *pb.Entry {
	index := sort.Search(len(l.Entries), func(i int) bool {
		return l.Entries[i].Term > term
	})
	var entry *pb.Entry
	if index > 0 && index < len(l.Entries) && l.Entries[index-1].Term == term {
		entry = l.Entries[index]
	} else {
		return nil
	}
	return entry
}

// isUpToDate
// Return true if self's log is less up to date than the args
func (l *raftLog) notUpToDate(index, term uint64) bool {
	lastEntry := l.LastEntry()
	if lastEntry.Term < term {
		return true
	}
	if lastEntry.Term == term && lastEntry.Index <= index {
		return true
	}
	return false
}

func (l *raftLog) sliceIndex(logIndex uint64) uint64 {
	return logIndex - l.IncludedIndex - 1
}
