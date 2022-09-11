package commands

func onCatJam(s *Session, i *Interaction) {

}

func init() {
	Command{
		Info: Descriptor{
			Name:        "catjam",
			Description: "Let's jam!",
		},
		Handler: onCatJam,
	}.add()
}
