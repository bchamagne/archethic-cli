package keychaincreatetransactionui

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"strconv"
	"strings"

	archethic "github.com/archethic-foundation/libgo"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type BackMsg bool

type CreateTransactionType int

type CreateTransactionMsg struct {
	ServiceName string
	Seed        string
	Url         string
}

var (
	focusedStyle               = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	blurredStyle               = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	cursorStyle                = focusedStyle.Copy()
	noStyle                    = lipgloss.NewStyle()
	helpStyle                  = blurredStyle.Copy()
	focusedAddAuthKey          = focusedStyle.Copy().Render("[ Add authorization key ]")
	blurredAddAuthKey          = fmt.Sprintf("[ %s ]", blurredStyle.Render("Add authorization key"))
	focusedLoadStorageNouncePK = focusedStyle.Copy().Render("[ Load Storage Nounce Public Key ]")
	blurredLoadStorageNouncePK = fmt.Sprintf("[ %s ]", blurredStyle.Render("Load Storage Nounce Public Key"))
	focusedButton              = focusedStyle.Copy().Render("[ Add ]")
	blurredButton              = fmt.Sprintf("[ %s ]", blurredStyle.Render("Add"))
	urlType                    = []string{"Local", "Testnet", "Mainnet", "Custom"}
	urls                       = map[string]string{
		"Local":   "http://localhost:4000",
		"Testnet": "https://testnet.archethic.net",
		"Mainnet": "https://mainnet.archethic.net",
		"Custom":  ""}
	transactionTypesList = []string{
		"Keychain Access",
		"Keychain",
		"Transfer",
		"Hosting",
		"Token",
		"Data",
		"Contract",
		"Code Proposal",
		"Code Approval",
	}
	transactionTypes = map[string]archethic.TransactionType{
		"Keychain Access": archethic.KeychainAccessType,
		"Keychain":        archethic.KeychainType,
		"Transfer":        archethic.TransferType,
		"Hosting":         archethic.HostingType,
		"Token":           archethic.TokenType,
		"Data":            archethic.DataType,
		"Contract":        archethic.ContractType,
		"Code Proposal":   archethic.CodeProposalType,
		"Code Approval":   archethic.CodeApprovalType,
	}
)

type RenderFunc func(m Model) string

type Model struct {
	Tabs                   []string
	TabContent             []RenderFunc
	activeTab              int
	focusInput             int
	mainInputs             []textinput.Model
	contentTextAreaInputs  []textarea.Model
	ucoInputs              []textinput.Model
	tokenInputs            []textinput.Model
	recipientsInput        textinput.Model
	ownershipsInputs       []textinput.Model
	transaction            archethic.TransactionBuilder
	secretKey              []byte
	authorizedKeys         []string
	storageNouncePublicKey string
	url                    string
	serviceName            string
	serviceMode            bool
	feedback               string
}

func New() Model {
	key := make([]byte, 32)
	rand.Read(key)
	m := Model{
		mainInputs:            make([]textinput.Model, 3),
		contentTextAreaInputs: make([]textarea.Model, 2),
		ucoInputs:             make([]textinput.Model, 2),
		tokenInputs:           make([]textinput.Model, 4),
		ownershipsInputs:      make([]textinput.Model, 2),
		focusInput:            0,
		activeTab:             0,
		transaction:           *archethic.NewTransaction(archethic.KeychainAccessType),
		secretKey:             key,
	}
	tabs := []string{"Main", "UCO Transfers", "Token Transfers", "Recipients", "Ownerships", "Content"}
	tabContent := []RenderFunc{main, ucoTransfer, tokenTransfer, recipients, ownerships, content}
	m.TabContent = tabContent
	m.Tabs = tabs

	for i := range m.mainInputs {
		t := textinput.New()
		t.CursorStyle = cursorStyle
		switch i {
		case 0:
			t.Prompt = ""
		case 1:
			t.Prompt = "> Access seed\n"
			t.EchoMode = textinput.EchoPassword
			t.EchoCharacter = '•'
		case 2:
			t.Prompt = ""
		}
		m.mainInputs[i] = t
	}

	for i := range m.ucoInputs {
		t := textinput.New()
		t.CursorStyle = cursorStyle
		switch i {
		case 0:
			t.Prompt = "> To:\n"
		case 1:
			t.Prompt = "> Amount:\n"
		}

		m.ucoInputs[i] = t
	}

	for i := range m.tokenInputs {
		t := textinput.New()
		t.CursorStyle = cursorStyle
		switch i {
		case 0:
			t.Prompt = "> To:\n"
		case 1:
			t.Prompt = "> Amount:\n"
		case 2:
			t.Prompt = "> Token Address:\n"
		case 3:
			t.Prompt = "> Token ID:\n"
		}

		m.tokenInputs[i] = t
	}

	m.recipientsInput = textinput.New()
	m.recipientsInput.CursorStyle = cursorStyle
	m.recipientsInput.Prompt = "> Recipient address:\n"

	for i := range m.ownershipsInputs {
		t := textinput.New()
		t.CursorStyle = cursorStyle
		switch i {
		case 0:
			t.Prompt = "> Secret:\n"
		case 1:
			t.Prompt = "> Authorization key:\n"
		}

		m.ownershipsInputs[i] = t
	}

	for i := range m.contentTextAreaInputs {
		ta := textarea.New()
		switch i {
		case 0:
			ta.Prompt = "> Content:\n"
		case 1:
			ta.Prompt = "> Code:\n"
		}
		m.contentTextAreaInputs[i] = ta
	}

	return m
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case CreateTransactionMsg:
		initParams := CreateTransactionMsg(msg)
		m.mainInputs[1].SetValue(initParams.Seed)
		m.mainInputs[0].SetValue(initParams.Url)
		m.url = initParams.Url
		m.serviceName = initParams.ServiceName
		m.serviceMode = m.serviceName != ""
		if m.serviceMode {
			m.focusInput = 6
		}
	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "esc":
			return m, func() tea.Msg {
				return BackMsg(true)
			}
		case "ctrl+c":
			return m, tea.Quit
		case "right", "tab":
			m.activeTab = min(m.activeTab+1, len(m.Tabs)-1)
			m.focusInput = 0
			return m, nil
		case "left", "shift+tab":
			m.activeTab = max(m.activeTab-1, 0)
			m.focusInput = 0
			return m, nil
		case "up", "down":
			m = getFocusIndex(m, keypress)
		case "enter":
			switch m.activeTab {
			case 0:

				if m.focusInput < 4 {
					u := urlType[m.focusInput]
					m.mainInputs[0].SetValue(urls[u])
					m.url = u
					m.focusInput = 4
				} else if m.focusInput > 5 && m.focusInput < 15 {
					t := transactionTypesList[m.focusInput-6]
					m.transaction.TxType = transactionTypes[t]
					m.focusInput = 15
				} else if m.focusInput == 15 {

					client := archethic.NewAPIClient(m.url)

					seed := archethic.MaybeConvertToHex(m.mainInputs[1].Value())

					keychain := archethic.GetKeychain(seed, *client)
					log.Println(keychain.Version)

					originPrivateKey, _ := hex.DecodeString("01019280BDB84B8F8AEDBA205FE3552689964A5626EE2C60AA10E3BF22A91A036009")

					m.transaction.Version = uint32(keychain.Version)

					genesisAddress := keychain.DeriveAddress(m.serviceName, 0)

					index := client.GetLastTransactionIndex(hex.EncodeToString(genesisAddress))

					m.transaction = keychain.BuildTransaction(m.transaction, m.serviceName, uint8(index))
					m.transaction.OriginSign(originPrivateKey)

					ts := archethic.NewTransactionSender(client)
					ts.AddOnSent(func() {
						m.feedback = "Transaction sent"
					})

					ts.AddOnError(func(sender, message string) {
						m.feedback = "Transaction error" + sender + message
					})

					ts.SendTransaction(&m.transaction, 1, 1000)

				}

			case 1:
				if m.focusInput == len(m.ucoInputs) {
					m = addUcoTransfer(m)
				}
			case 2:
				if m.focusInput == len(m.tokenInputs) {
					m = addTokenTransfer(m)
				}
			case 3:
				if m.focusInput == 1 || m.focusInput == 0 {
					m = addRecipient(m)
				}
			case 4:
				switch m.focusInput {
				case len(m.ownershipsInputs) + len(m.authorizedKeys):
					m = addAuthorizedKey(m)
				case len(m.ownershipsInputs) + len(m.authorizedKeys) + 1:
					m = loadStorageNouncePublicKey(m)
				case len(m.ownershipsInputs) + len(m.authorizedKeys) + 2:
					m = addOwnership(m)
				}
			}
		case "d":
			switch m.activeTab {
			case 1:
				if m.focusInput > len(m.ucoInputs) {
					m = deleteUcoTransfer(m)
				}
			case 2:
				if m.focusInput > len(m.tokenInputs) {
					m = deleteTokenTransfer(m)
				}
			case 3:
				if m.focusInput > 1 {
					m = deleteRecipient(m)
				}
			case 4:
				if m.focusInput > len(m.ownershipsInputs)-1 && m.focusInput < len(m.ownershipsInputs)+len(m.authorizedKeys) {
					m = deleteAuthorizedKey(m)
				} else if m.focusInput > len(m.ownershipsInputs)+len(m.authorizedKeys)+2 {
					m = deleteOwnership(m)
				}
			}
		}
	}
	m, cmds := updateFocus(m)

	cmds = append(cmds, m.updateInputs(msg)...)

	return m, tea.Batch(cmds...)
}

func (m *Model) updateInputs(msg tea.Msg) []tea.Cmd {
	var cmds []tea.Cmd
	switch m.activeTab {
	case 0:
		cmds := make([]tea.Cmd, len(m.mainInputs))
		for i := range m.mainInputs {
			m.mainInputs[i], cmds[i] = m.mainInputs[i].Update(msg)
		}
	case 1:
		cmds := make([]tea.Cmd, len(m.ucoInputs))
		for i := range m.ucoInputs {
			m.ucoInputs[i], cmds[i] = m.ucoInputs[i].Update(msg)
		}
	case 2:
		cmds := make([]tea.Cmd, len(m.tokenInputs))
		for i := range m.tokenInputs {
			m.tokenInputs[i], cmds[i] = m.tokenInputs[i].Update(msg)
		}
	case 3:
		cmds := make([]tea.Cmd, 1)
		m.recipientsInput, cmds[0] = m.recipientsInput.Update(msg)
	case 4:
		cmds := make([]tea.Cmd, len(m.ownershipsInputs))
		for i := range m.ownershipsInputs {
			m.ownershipsInputs[i], cmds[i] = m.ownershipsInputs[i].Update(msg)
		}
	case 5:
		cmds := make([]tea.Cmd, len(m.contentTextAreaInputs))
		for i := range m.contentTextAreaInputs {
			m.contentTextAreaInputs[i], cmds[i] = m.contentTextAreaInputs[i].Update(msg)
		}
	}

	return cmds
}

func updateFocus(m Model) (Model, []tea.Cmd) {
	var cmds []tea.Cmd
	switch m.activeTab {
	case 0:
		cmds := make([]tea.Cmd, len(m.mainInputs))
		for i := 0; i <= len(m.mainInputs)-1; i++ {
			if i == m.focusInput-4 {
				// Set focused state
				cmds[i] = m.mainInputs[i].Focus()
				continue
			}
			// Remove focused state
			m.mainInputs[i].Blur()
			m.mainInputs[i].PromptStyle = noStyle
			m.mainInputs[i].TextStyle = noStyle
		}

	case 1:

		cmds := make([]tea.Cmd, len(m.ucoInputs))
		for i := 0; i <= len(m.ucoInputs)-1; i++ {
			if i == m.focusInput {
				// Set focused state
				cmds[i] = m.ucoInputs[i].Focus()
				continue
			}
			// Remove focused state
			m.ucoInputs[i].Blur()
			m.ucoInputs[i].PromptStyle = noStyle
			m.ucoInputs[i].TextStyle = noStyle
		}

	case 2:

		cmds := make([]tea.Cmd, len(m.tokenInputs))
		for i := 0; i <= len(m.tokenInputs)-1; i++ {
			if i == m.focusInput {
				// Set focused state
				cmds[i] = m.tokenInputs[i].Focus()
				continue
			}
			// Remove focused state
			m.tokenInputs[i].Blur()
			m.tokenInputs[i].PromptStyle = noStyle
			m.tokenInputs[i].TextStyle = noStyle
		}

	case 3:
		if m.focusInput == 0 {
			cmds = append(cmds, m.recipientsInput.Focus())
			m.recipientsInput.PromptStyle = focusedStyle
			m.recipientsInput.TextStyle = focusedStyle
		} else {
			m.recipientsInput.Blur()
			m.recipientsInput.PromptStyle = noStyle
			m.recipientsInput.TextStyle = noStyle
		}
	case 4:
		cmds := make([]tea.Cmd, len(m.ownershipsInputs))
		for i := 0; i <= len(m.ownershipsInputs)-1; i++ {
			if i == m.focusInput {
				// Set focused state
				cmds[i] = m.ownershipsInputs[i].Focus()
				continue
			}
			// Remove focused state
			m.ownershipsInputs[i].Blur()
			m.ownershipsInputs[i].PromptStyle = noStyle
			m.ownershipsInputs[i].TextStyle = noStyle
		}
	case 5:
		cmds := make([]tea.Cmd, len(m.contentTextAreaInputs))
		for i := 0; i <= len(m.contentTextAreaInputs)-1; i++ {
			if i == m.focusInput {
				// Set focused state
				cmds[i] = m.contentTextAreaInputs[i].Focus()
				continue
			}
			// Remove focused state
			m.contentTextAreaInputs[i].Blur()
		}

	}
	return m, cmds
}

func getFocusIndex(m Model, keypress string) Model {
	if keypress == "up" {
		m.focusInput--
	} else {
		m.focusInput++
	}
	switch m.activeTab {
	case 0:
		if m.serviceMode {
			if m.focusInput > 15 {
				m.focusInput = 6
			} else if m.focusInput < 6 {
				m.focusInput = 15
			}
		} else {
			if m.focusInput > 15 {
				m.focusInput = 0
			} else if m.focusInput < 0 {
				m.focusInput = 15
			}
		}

	case 1:
		if m.focusInput > len(m.ucoInputs)+len(m.transaction.Data.Ledger.Uco.Transfers) {
			m.focusInput = 0
		} else if m.focusInput < 0 {
			m.focusInput = len(m.ucoInputs) + len(m.transaction.Data.Ledger.Uco.Transfers)
		}
	case 2:
		if m.focusInput > len(m.tokenInputs)+len(m.transaction.Data.Ledger.Token.Transfers) {
			m.focusInput = 0
		} else if m.focusInput < 0 {
			m.focusInput = len(m.tokenInputs) + len(m.transaction.Data.Ledger.Token.Transfers)
		}
	case 3:
		if m.focusInput > 1+1+len(m.transaction.Data.Recipients) {
			m.focusInput = 0
		} else if m.focusInput < 0 {
			m.focusInput = 1 + 1 + len(m.transaction.Data.Recipients)
		}
	case 4:
		if m.focusInput > len(m.ownershipsInputs)+len(m.authorizedKeys)+len(m.transaction.Data.Ownerships)+2 {
			m.focusInput = 0
		} else if m.focusInput < 0 {
			m.focusInput = len(m.ownershipsInputs) + len(m.authorizedKeys) + len(m.transaction.Data.Ownerships) + 2
		}
	case 5:
		if m.focusInput > len(m.contentTextAreaInputs) {
			m.focusInput = 0
		} else if m.focusInput < 0 {
			m.focusInput = len(m.contentTextAreaInputs)
		}
	}
	return m
}

func tabBorderWithBottom(left, middle, right string) lipgloss.Border {
	border := lipgloss.RoundedBorder()
	border.BottomLeft = left
	border.Bottom = middle
	border.BottomRight = right
	return border
}

var (
	inactiveTabBorder = tabBorderWithBottom("┴", "─", "┴")
	activeTabBorder   = tabBorderWithBottom("┘", " ", "└")
	docStyle          = lipgloss.NewStyle().Padding(1, 2, 1, 2)
	highlightColor    = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	inactiveTabStyle  = lipgloss.NewStyle().Border(inactiveTabBorder, true).BorderForeground(highlightColor).Padding(0, 1)
	activeTabStyle    = inactiveTabStyle.Copy().Border(activeTabBorder, true)
	windowStyle       = lipgloss.NewStyle().BorderForeground(highlightColor).Padding(2, 0).Align(lipgloss.Left).Border(lipgloss.NormalBorder()).UnsetBorderTop()
)

func (m Model) View() string {
	doc := strings.Builder{}

	var renderedTabs []string

	for i, t := range m.Tabs {
		var style lipgloss.Style
		isFirst, isLast, isActive := i == 0, i == len(m.Tabs)-1, i == m.activeTab
		if isActive {
			style = activeTabStyle.Copy()
		} else {
			style = inactiveTabStyle.Copy()
		}
		border, _, _, _, _ := style.GetBorder()
		if isFirst && isActive {
			border.BottomLeft = "│"
		} else if isFirst && !isActive {
			border.BottomLeft = "├"
		} else if isLast && isActive {
			border.BottomRight = "│"
		} else if isLast && !isActive {
			border.BottomRight = "┤"
		}
		style = style.Border(border)
		renderedTabs = append(renderedTabs, style.Render(t))
	}

	row := lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...)
	doc.WriteString(row)
	doc.WriteString("\n")
	doc.WriteString(windowStyle.Width((lipgloss.Width(row) - windowStyle.GetHorizontalFrameSize())).Render(m.TabContent[m.activeTab](m)))
	return docStyle.Render(doc.String())
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func addUcoTransfer(m Model) Model {
	toHex := m.ucoInputs[0].Value()
	to, err := hex.DecodeString(toHex)
	if err != nil {
		panic(err)
	}
	amountStr := m.ucoInputs[1].Value()
	amount, err := strconv.ParseUint(amountStr, 10, 64)
	if err != nil {
		panic(err)
	}
	m.transaction.AddUcoTransfer(to, amount)
	m.ucoInputs[0].SetValue("")
	m.ucoInputs[1].SetValue("")
	return m
}

func deleteUcoTransfer(m Model) Model {
	focusIndex := m.focusInput - len(m.ucoInputs) - 1
	m.transaction.Data.Ledger.Uco.Transfers = append(m.transaction.Data.Ledger.Uco.Transfers[:focusIndex], m.transaction.Data.Ledger.Uco.Transfers[focusIndex+1:]...)
	m.focusInput--
	return m
}

func addTokenTransfer(m Model) Model {
	toHex := m.tokenInputs[0].Value()
	to, err := hex.DecodeString(toHex)
	if err != nil {
		panic(err)
	}
	amountStr := m.tokenInputs[1].Value()
	amount, err := strconv.ParseUint(amountStr, 10, 64)
	if err != nil {
		panic(err)
	}
	tokenAddressHex := m.tokenInputs[2].Value()
	tokenAddress, err := hex.DecodeString(tokenAddressHex)
	if err != nil {
		panic(err)
	}
	tokenIdStr := m.tokenInputs[3].Value()

	tokenId, err := strconv.Atoi(tokenIdStr)
	if err != nil {
		panic(err)
	}
	m.transaction.AddTokenTransfer(to, tokenAddress, amount, tokenId)
	m.tokenInputs[0].SetValue("")
	m.tokenInputs[1].SetValue("")
	m.tokenInputs[2].SetValue("")
	m.tokenInputs[3].SetValue("")
	return m
}

func deleteTokenTransfer(m Model) Model {
	focusIndex := m.focusInput - len(m.tokenInputs) - 1
	m.transaction.Data.Ledger.Token.Transfers = append(m.transaction.Data.Ledger.Token.Transfers[:focusIndex], m.transaction.Data.Ledger.Token.Transfers[focusIndex+1:]...)
	m.focusInput--
	return m
}

func addRecipient(m Model) Model {
	recipientHex := m.recipientsInput.Value()
	recipient, err := hex.DecodeString(recipientHex)
	if err != nil {
		panic(err)
	}
	m.transaction.AddRecipient(recipient)
	m.recipientsInput.SetValue("")
	return m
}

func deleteRecipient(m Model) Model {
	focusIndex := m.focusInput - 1 - 1
	m.transaction.Data.Recipients = append(m.transaction.Data.Recipients[:focusIndex], m.transaction.Data.Recipients[focusIndex+1:]...)
	m.focusInput--
	return m
}

func addOwnership(m Model) Model {

	secret := m.ownershipsInputs[0].Value()

	if m.ownershipsInputs[1].Value() != "" {
		m = addAuthorizedKey(m)
	}

	cipher := archethic.AesEncrypt([]byte(secret), m.secretKey)
	authorizedKeys := make([]archethic.AuthorizedKey, len(m.authorizedKeys))
	for i, key := range m.authorizedKeys {
		keyByte, err := hex.DecodeString(key)
		if err != nil {
			panic(err)
		}
		authorizedKeys[i] = archethic.AuthorizedKey{
			PublicKey:          keyByte,
			EncryptedSecretKey: archethic.EcEncrypt(m.secretKey, keyByte),
		}
	}
	m.transaction.AddOwnership(cipher, authorizedKeys)
	m.authorizedKeys = []string{}
	m.ownershipsInputs[0].SetValue("")
	m.ownershipsInputs[1].SetValue("")
	return m
}

func deleteOwnership(m Model) Model {
	focusIndex := m.focusInput - len(m.ownershipsInputs) - len(m.authorizedKeys) - 3
	m.transaction.Data.Ownerships = append(m.transaction.Data.Ownerships[:focusIndex], m.transaction.Data.Ownerships[focusIndex+1:]...)
	m.focusInput--
	return m
}

func addAuthorizedKey(m Model) Model {
	authorizedKey := m.ownershipsInputs[1].Value()
	m.authorizedKeys = append(m.authorizedKeys, authorizedKey)
	m.ownershipsInputs[1].SetValue("")
	return m
}

func deleteAuthorizedKey(m Model) Model {
	focusIndex := m.focusInput - len(m.ownershipsInputs)
	m.authorizedKeys = append(m.authorizedKeys[:focusIndex], m.authorizedKeys[focusIndex+1:]...)
	m.focusInput--
	return m
}

func loadStorageNouncePublicKey(m Model) Model {

	if m.storageNouncePublicKey == "" {
		client := archethic.NewAPIClient(m.url)
		m.storageNouncePublicKey = client.GetStorageNoncePublicKey()
	}

	m.ownershipsInputs[1].SetValue(m.storageNouncePublicKey)
	return m
}

func main(m Model) string {
	var b strings.Builder
	if m.serviceMode {
		b.WriteString("Creating transaction for service " + m.serviceName + "\n\n")
	} else {
		b.WriteString("> Node endpoint:\n")
		b.WriteString(urlView(m))
		b.WriteString(m.mainInputs[0].View() + "\n\n")
		b.WriteString(m.mainInputs[1].View() + "\n\n")
	}

	b.WriteString("> Transaction type:\n")
	b.WriteString(transactionTypeView(m))

	button := &blurredButton
	if m.focusInput == 15 {
		button = &focusedButton
	}
	fmt.Fprintf(&b, "\n\n%s\n\n", *button)

	b.WriteString(m.feedback)
	return b.String()
}

func urlView(m Model) string {
	s := strings.Builder{}

	for i := 0; i < len(urlType); i++ {
		var u string
		if m.url == urlType[i] {
			u = "(•) "
		} else {
			u = "( ) "
		}
		u += urlType[i]
		if i == m.focusInput {
			s.WriteString(focusedStyle.Render(u))
		} else {
			s.WriteString(u)
		}
		s.WriteString("\n")
	}

	return s.String()
}

func transactionTypeView(m Model) string {
	s := strings.Builder{}

	for i, t := range transactionTypesList {
		var u string
		if m.transaction.TxType == transactionTypes[t] {
			u = "(•) "
		} else {
			u = "( ) "
		}
		u += t
		if m.focusInput == i+6 {
			s.WriteString(focusedStyle.Render(u))
		} else {
			s.WriteString(u)
		}
		s.WriteString("\n")
	}

	return s.String()
}

func ucoTransfer(m Model) string {
	var b strings.Builder
	for i := range m.ucoInputs {
		b.WriteString(m.ucoInputs[i].View())
		if i < len(m.ucoInputs)-1 {
			b.WriteRune('\n')
		}
	}
	button := &blurredButton
	if m.focusInput == len(m.ucoInputs) {
		button = &focusedButton
	}
	fmt.Fprintf(&b, "\n\n%s\n\n", *button)

	startCount := len(m.ucoInputs) + 1 // +1 for the button
	for i, t := range m.transaction.Data.Ledger.Uco.Transfers {
		transfer := fmt.Sprintf("%s: %d\n", hex.EncodeToString(t.To), t.Amount)
		if m.focusInput == startCount+i {
			b.WriteString(focusedStyle.Render(transfer))
			continue
		} else {
			b.WriteString(transfer)
		}
	}
	return b.String()
}

func tokenTransfer(m Model) string {
	var b strings.Builder
	for i := range m.tokenInputs {
		b.WriteString(m.tokenInputs[i].View())
		if i < len(m.tokenInputs)-1 {
			b.WriteRune('\n')
		}
	}
	button := &blurredButton
	if m.focusInput == len(m.tokenInputs) {
		button = &focusedButton
	}
	fmt.Fprintf(&b, "\n\n%s\n\n", *button)

	startCount := len(m.tokenInputs) + 1 // +1 for the button
	for i, t := range m.transaction.Data.Ledger.Token.Transfers {
		transfer := fmt.Sprintf("%s : %d - %s %d \n", hex.EncodeToString(t.To), t.Amount, hex.EncodeToString(t.TokenAddress), t.TokenId)
		if m.focusInput == startCount+i {
			b.WriteString(focusedStyle.Render(transfer))
			continue
		} else {
			b.WriteString(transfer)
		}
	}
	return b.String()
}

func recipients(m Model) string {
	var b strings.Builder
	b.WriteString(m.recipientsInput.View())
	b.WriteRune('\n')

	button := &blurredButton
	if m.focusInput == 1 {
		button = &focusedButton
	}
	fmt.Fprintf(&b, "\n\n%s\n\n", *button)

	startCount := 1 + 1 // +1 for the button
	for i, t := range m.transaction.Data.Recipients {
		recipient := fmt.Sprintf("%s\n", hex.EncodeToString(t))
		if m.focusInput == startCount+i {
			b.WriteString(focusedStyle.Render(recipient))
			continue
		} else {
			b.WriteString(recipient)
		}
	}
	return b.String()
}

func ownerships(m Model) string {
	var b strings.Builder
	for i := range m.ownershipsInputs {
		b.WriteString(m.ownershipsInputs[i].View())
		if i < len(m.ownershipsInputs)-1 {
			b.WriteRune('\n')
		}
	}

	if len(m.authorizedKeys) > 0 {
		b.WriteString("\nList of authorized keys to add:\n")
		for i := range m.authorizedKeys {
			if m.focusInput == len(m.ownershipsInputs)+i {
				b.WriteString(focusedStyle.Render(m.authorizedKeys[i]))
			} else {
				b.WriteString(m.authorizedKeys[i])
			}
			b.WriteRune('\n')
		}
	}

	buttonAddAuthKey := &blurredAddAuthKey
	if m.focusInput == len(m.ownershipsInputs)+len(m.authorizedKeys) {
		buttonAddAuthKey = &focusedAddAuthKey
	}
	fmt.Fprintf(&b, "\n\n%s", *buttonAddAuthKey)

	buttonLoadStorageNouncePK := &blurredLoadStorageNouncePK
	if m.focusInput == len(m.ownershipsInputs)+len(m.authorizedKeys)+1 {
		buttonLoadStorageNouncePK = &focusedLoadStorageNouncePK
	}
	fmt.Fprintf(&b, "\n\n%s", *buttonLoadStorageNouncePK)

	button := &blurredButton
	if m.focusInput == len(m.ownershipsInputs)+len(m.authorizedKeys)+2 {
		button = &focusedButton
	}
	fmt.Fprintf(&b, "\n\n%s\n\n", *button)

	startCount := len(m.ownershipsInputs) + len(m.authorizedKeys) + 3
	for i, o := range m.transaction.Data.Ownerships {
		ownerships := "**** "
		for j := range o.AuthorizedKeys {
			keyHex := hex.EncodeToString(o.AuthorizedKeys[j].PublicKey)
			ownerships += fmt.Sprintf("%s\n", keyHex)
		}
		if m.focusInput == startCount+i {
			b.WriteString(focusedStyle.Render(ownerships))
			continue
		} else {
			b.WriteString(ownerships)
		}
	}
	return b.String()
}

func content(m Model) string {
	var b strings.Builder
	for i := range m.contentTextAreaInputs {
		b.WriteString(m.contentTextAreaInputs[i].View())
	}

	return b.String()
}
