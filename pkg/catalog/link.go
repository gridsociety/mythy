package catalog

// linkMenuToMessages walks the MENU tree and resolves every <DATA NAME=>
// and <COMMAND NAME=> to the matching <message name=> in the WSDL layer.
// DATA/COMMAND entries that don't match a message are left with Message
// = nil; that's not an error (some hierarchical DATA leaves don't
// correspond to a register on their own).
func linkMenuToMessages(tpl *Template) {
	if tpl.Menu == nil {
		return
	}
	var walk func(*Group)
	walk = func(g *Group) {
		for _, d := range g.Data {
			d.Message = tpl.Messages[d.Name]
		}
		for _, c := range g.Commands {
			c.Message = tpl.Messages[c.Name]
		}
		for _, c := range g.Children {
			walk(c)
		}
	}
	walk(tpl.Menu)
}
