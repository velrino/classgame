package contentpolicy

import (
	"strings"
	"unicode"
)

type Violation string

const (
	ViolationNone            Violation = ""
	ViolationAbusiveLanguage Violation = "abusive_language"
	ViolationUnsafeIntent    Violation = "unsafe_intent"
	ViolationRoleOverride    Violation = "role_override"
)

type Decision struct {
	Allowed   bool
	Violation Violation
	Message   string
}

var abusiveTokens = newTokenSet([]string{
	"arrombado", "arrombada", "asshole", "babaca", "bastard", "bastardo",
	"bastarda", "bitch", "bullshit", "burro", "burra", "cacete", "caralho",
	"cunt", "cuzao", "desgracado", "desgracada", "dick", "dumbass", "fag",
	"faggot", "fdp", "foda", "foder", "fuck", "fucked", "fucker", "fucking",
	"idiot", "idiota", "idiotas",
	"imbecil", "imbecis", "kike", "merda", "moron", "motherfucker", "nigga",
	"nigger", "otario", "otaria", "porra", "puta", "puto", "retard",
	"retardado", "retardada", "retarded", "shit", "slut", "spic", "stupid",
	"tranny", "vagabunda", "vagabundo", "viado", "whore",
})

var abusivePhrases = []string{
	"filha da puta",
	"filho da puta",
	"fuck you",
	"go fuck yourself",
	"go die",
	"mother fucker",
	"pau no cu",
	"se foda",
	"vai morrer",
	"vai se foder",
}

var unsafeTokens = newTokenSet([]string{
	"ddos", "malware", "phishing", "porn", "pornografia", "pornography",
	"ransomware", "suicide", "suicidio",
})

var unsafePhrases = []string{
	"build a bomb",
	"credential theft",
	"create malware",
	"exploit a child",
	"generate malware",
	"kill a student",
	"kill students",
	"kill the teacher",
	"make a bomb",
	"matar aluno",
	"matar alunos",
	"matar professor",
	"matar professora",
	"roubar credenciais",
	"roubar senhas",
	"school shooting",
	"self harm",
	"sexual content",
	"shoot a student",
	"shoot students",
	"steal credentials",
	"steal passwords",
}

var roleOverridePhrases = []string{
	"bypass policy",
	"ignore all instructions",
	"ignore previous instructions",
	"ignore system instructions",
	"ignore suas instrucoes",
	"ignorar instrucoes anteriores",
	"reveal developer message",
	"reveal system prompt",
	"revele o prompt do sistema",
	"system prompt",
}

var educationalTokens = newTokenSet([]string{
	"adicao", "algebra", "aluno", "alunos", "analisar", "analyze", "aplicar",
	"apply", "angle", "angulo", "aprender", "aprendizagem", "arithmetic", "art",
	"arte", "assess", "assessment", "avaliacao", "avaliar", "astronomia",
	"astronomy", "atividade", "atividades", "aula", "biology",
	"biologia", "calculate", "calcular", "capitulo", "chapter", "chemistry",
	"ciencia", "ciencias", "class", "classificar", "classify", "classroom",
	"coding", "compare", "comparar", "compreender", "conceito", "concept",
	"conhecimento", "decimal", "decimais", "demonstrar", "demonstrate",
	"descrever", "describe", "discuss", "discutir", "division", "educacao",
	"education", "english", "ensinar", "entender", "equacao", "equation",
	"escola", "escrever", "escrita", "estudante", "estudantes", "estudar",
	"evaluate", "exercise", "exercicio", "exercicios", "explain", "explicar",
	"feedback", "fisica",
	"fraction", "fractions", "fracao", "fracoes", "geografia", "geography",
	"geometry", "geometria", "grade", "grammar", "gramatica", "habilidade",
	"historia", "history", "identify", "identificar", "knowledge", "language",
	"learn", "learning", "lesson", "ler", "leitura", "literatura", "literature",
	"math", "mathematics", "matematica", "multiplicacao", "multiplication",
	"music", "musica", "number", "numero", "objetivo", "objective", "observar",
	"observe", "pagina", "page", "percent",
	"percentage", "physics", "portugues", "practice", "praticar", "professor",
	"professora", "question", "quiz", "raciocinio", "read", "reading",
	"reasoning", "recall", "reconhecer", "recognize", "responder", "response",
	"review", "revisar", "school", "science", "skill", "solve", "student",
	"students", "study", "subject", "summarize", "resumir",
	"subtracao", "subtraction", "teacher", "tense", "turma", "understand",
	"unit", "worksheet", "write", "writing",
})

var educationalPhrases = []string{
	"computer science",
	"grade level",
	"language arts",
	"learning goal",
	"nivel escolar",
	"sala de aula",
	"serie escolar",
	"social studies",
}

var textFoldReplacer = strings.NewReplacer(
	"á", "a", "à", "a", "â", "a", "ã", "a", "ä", "a",
	"é", "e", "è", "e", "ê", "e", "ë", "e",
	"í", "i", "ì", "i", "î", "i", "ï", "i",
	"ó", "o", "ò", "o", "ô", "o", "õ", "o", "ö", "o",
	"ú", "u", "ù", "u", "û", "u", "ü", "u",
	"ç", "c", "ñ", "n", "@", "a", "$", "s",
)

var leetReplacer = strings.NewReplacer(
	"0", "o",
	"1", "i",
	"3", "e",
	"4", "a",
	"5", "s",
	"7", "t",
	"8", "b",
)

// ReviewEducationalText rejects abusive, unsafe, and role-changing content
// without calling a model. It is intentionally field-agnostic so handlers can
// reuse the same policy for any teacher- or student-authored input.
func ReviewEducationalText(value string) Decision {
	normalized := normalize(value)
	if normalized == "" {
		return Decision{Allowed: true}
	}
	normalized = strings.Join(mergeSingleLetterRuns(strings.Fields(normalized)), " ")

	if containsBlockedToken(normalized, abusiveTokens) || containsAnyPhrase(normalized, abusivePhrases) {
		return Decision{
			Allowed:   false,
			Violation: ViolationAbusiveLanguage,
			Message:   "Use respectful, grade-appropriate educational language.",
		}
	}
	if containsBlockedToken(normalized, unsafeTokens) || containsAnyPhrase(normalized, unsafePhrases) {
		return Decision{
			Allowed:   false,
			Violation: ViolationUnsafeIntent,
			Message:   "Only safe, grade-appropriate educational content is allowed.",
		}
	}
	if containsAnyPhrase(normalized, roleOverridePhrases) {
		return Decision{
			Allowed:   false,
			Violation: ViolationRoleOverride,
			Message:   "Instructions that try to change the educational agent's role are not allowed.",
		}
	}

	return Decision{Allowed: true}
}

// HasEducationalContext checks for an explicit learning signal across one or
// more fields. Callers can pass a complete form or only the fields they own.
func HasEducationalContext(values ...string) bool {
	normalized := normalize(strings.Join(values, " "))
	if normalized == "" {
		return false
	}

	for _, token := range strings.Fields(normalized) {
		if _, exists := educationalTokens[canonicalToken(token)]; exists {
			return true
		}
	}

	return containsAnyPhrase(normalized, educationalPhrases)
}

func containsBlockedToken(normalized string, blocked map[string]struct{}) bool {
	for _, token := range mergeSingleLetterRuns(strings.Fields(normalized)) {
		canonical := canonicalToken(token)
		if _, exists := blocked[canonical]; exists {
			return true
		}
		if hasExtendedRepeat(canonical) {
			collapsed := collapseRepeatedRunes(canonical)
			if _, exists := blocked[collapsed]; exists {
				return true
			}
			for blockedToken := range blocked {
				if collapsed == collapseRepeatedRunes(blockedToken) {
					return true
				}
			}
		}
	}

	return false
}

func containsAnyPhrase(normalized string, phrases []string) bool {
	normalized = canonicalToken(normalized)
	padded := " " + normalized + " "
	compact := strings.ReplaceAll(normalized, " ", "")
	for _, phrase := range phrases {
		if strings.Contains(padded, " "+phrase+" ") {
			return true
		}
		if strings.Contains(compact, strings.ReplaceAll(phrase, " ", "")) {
			return true
		}
	}

	if !hasExtendedRepeat(normalized) {
		return false
	}

	collapsed := collapseRepeatedRunes(normalized)
	for _, phrase := range phrases {
		if strings.Contains(" "+collapsed+" ", " "+collapseRepeatedRunes(phrase)+" ") {
			return true
		}
	}

	return false
}

func newTokenSet(values []string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		result[canonicalToken(normalize(value))] = struct{}{}
	}

	return result
}

func canonicalToken(value string) string {
	return leetReplacer.Replace(value)
}

func collapseRepeatedRunes(value string) string {
	var result strings.Builder
	var previous rune
	for _, current := range value {
		if current == previous {
			continue
		}
		result.WriteRune(current)
		previous = current
	}

	return result.String()
}

func hasExtendedRepeat(value string) bool {
	var previous rune
	repeatCount := 0
	for _, current := range value {
		if current == previous {
			repeatCount++
			if repeatCount >= 3 {
				return true
			}
			continue
		}
		previous = current
		repeatCount = 1
	}

	return false
}

func mergeSingleLetterRuns(tokens []string) []string {
	result := make([]string, 0, len(tokens))
	for index := 0; index < len(tokens); {
		end := index
		for end < len(tokens) && len([]rune(tokens[end])) == 1 {
			end++
		}
		if end-index >= 3 {
			result = append(result, strings.Join(tokens[index:end], ""))
			index = end
			continue
		}

		result = append(result, tokens[index])
		index++
	}

	return result
}

func normalize(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = textFoldReplacer.Replace(value)

	var result strings.Builder
	result.Grow(len(value))
	for _, current := range value {
		if unicode.IsLetter(current) || unicode.IsDigit(current) {
			result.WriteRune(current)
			continue
		}
		result.WriteByte(' ')
	}

	return strings.Join(strings.Fields(result.String()), " ")
}
