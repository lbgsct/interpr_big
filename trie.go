package main


// Узел дерева
type Node struct {
	children    map[rune]*Node
	value       interface{}
	isEndOfWord bool
}

func NewNode() *Node {
	return &Node{
		children: make(map[rune]*Node),
	}
}

// Дерево
type Trie struct {
	root *Node
}

//cоздание
func NewTrie() *Trie {
	return &Trie{
		root: NewNode(),
	}
}

// Вставка
func (t *Trie) Insert(key string, value interface{}) {
	node := t.root
	for _, char := range key {
		if _, found := node.children[char]; !found {
			node.children[char] = NewNode()
		}
		node = node.children[char]
	}
	node.isEndOfWord = true
	node.value = value
}

// Поиск
func (t *Trie) Search(key string) interface{} {
	node := t.root
	for _, char := range key {
		if _, found := node.children[char]; !found {
			return nil
		}
		node = node.children[char]
	}
	if node.isEndOfWord {
		return node.value
	}
	return nil
}

// Удаление
func (t *Trie) Delete(key string) {
	var delet func(node *Node, key string, depth int) bool
	delet = func(node *Node, key string, depth int) bool {
		if node == nil {
			return false
		}

		if depth == len(key) {
			if node.isEndOfWord {
				node.isEndOfWord = false
			}
			return len(node.children) == 0
		}

		char := rune(key[depth])
		if delet(node.children[char], key, depth+1) {
			delete(node.children, char)
			return len(node.children) == 0 && !node.isEndOfWord
		}

		return false
	}

	delet(t.root, key, 0)
}

// Получение всех ключей
func (t *Trie) ObtainAll() []string {
	var results []string
	var obtainAll func(node *Node, prefix string)
	obtainAll = func(node *Node, prefix string) {
		if node == nil {
			return
		}
		if node.isEndOfWord {
			results = append(results, prefix)
		}
		for char, nextNode := range node.children {
			obtainAll(nextNode, prefix+string(char))
		}
	}

	obtainAll(t.root, "")
	return results
}
