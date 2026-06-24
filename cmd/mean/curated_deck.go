package main

import "github.com/umang/mean-cli/internal/models"

var curatedDeck = []models.Word{
	{
		Word:          "ephemeral",
		Pronunciation: "/ɪˈfɛm.ər.əl/",
		Definitions: []models.Definition{
			{PartOfSpeech: "adjective", Meaning: "Lasting for a very short time; transient.", Example: "Fame in the age of social media is often ephemeral."},
		},
		Synonyms:  []string{"transient", "fleeting", "momentary", "short-lived"},
		Antonyms:  []string{"eternal", "permanent", "enduring"},
		ExamLevel: "GRE/Advanced",
	},
	{
		Word:          "serendipity",
		Pronunciation: "/ˌsɛr.ənˈdɪp.ɪ.ti/",
		Definitions: []models.Definition{
			{PartOfSpeech: "noun", Meaning: "The occurrence of valuable events by chance in a happy or beneficial way.", Example: "We found the charming little restaurant by pure serendipity."},
		},
		Synonyms:  []string{"fluke", "chance", "coincidence", "happy accident"},
		Antonyms:  []string{"misfortune", "bad luck"},
		ExamLevel: "GRE/Advanced",
	},
	{
		Word:          "quixotic",
		Pronunciation: "/kwɪkˈsɒt.ɪk/",
		Definitions: []models.Definition{
			{PartOfSpeech: "adjective", Meaning: "Exceedingly idealistic, unrealistic, and impractical.", Example: "He launched a quixotic campaign to reform the entire system overnight."},
		},
		Synonyms:  []string{"idealistic", "visionary", "impractical", "unrealistic"},
		Antonyms:  []string{"practical", "pragmatic", "realistic"},
		ExamLevel: "GRE/Advanced",
	},
	{
		Word:          "nefarious",
		Pronunciation: "/nɪˈfɛə.ri.əs/",
		Definitions: []models.Definition{
			{PartOfSpeech: "adjective", Meaning: "Wicked, impious, or criminal; extremely heinous.", Example: "The villain hatched a nefarious plot to take over the city's power grid."},
		},
		Synonyms:  []string{"wicked", "evil", "villainous", "heinous"},
		Antonyms:  []string{"noble", "admirable", "righteous"},
		ExamLevel: "GRE/Advanced",
	},
	{
		Word:          "verbose",
		Pronunciation: "/vɜːˈbəʊs/",
		Definitions: []models.Definition{
			{PartOfSpeech: "adjective", Meaning: "Using or expressed in more words than are needed.", Example: "The senator's speech was verbose and circular, putting many to sleep."},
		},
		Synonyms:  []string{"wordy", "loquacious", "garrulous", "prolix"},
		Antonyms:  []string{"terse", "succinct", "laconic"},
		ExamLevel: "IELTS/TOEFL",
	},
	{
		Word:          "terse",
		Pronunciation: "/tɜːs/",
		Definitions: []models.Definition{
			{PartOfSpeech: "adjective", Meaning: "Sparing in the use of words; abrupt or concise.", Example: "She dismissed his excuses with a terse 'No.'"},
		},
		Synonyms:  []string{"concise", "brief", "succinct", "short"},
		Antonyms:  []string{"verbose", "wordy", "rambling"},
		ExamLevel: "intermediate",
	},
	{
		Word:          "lucid",
		Pronunciation: "/ˈluː.sɪd/",
		Definitions: []models.Definition{
			{PartOfSpeech: "adjective", Meaning: "Expressed clearly; easy to understand; rational or sane.", Example: "The professor gave a lucid explanation of complex quantum mechanics."},
		},
		Synonyms:  []string{"clear", "coherent", "understandable", "rational"},
		Antonyms:  []string{"confusing", "muddled", "obscure"},
		ExamLevel: "intermediate",
	},
	{
		Word:          "languid",
		Pronunciation: "/ˈlæŋ.ɡwɪd/",
		Definitions: []models.Definition{
			{PartOfSpeech: "adjective", Meaning: "Displaying or having a disinclination for physical exertion or effort; slow and relaxed.", Example: "They spent a languid Sunday afternoon reading under the oak tree."},
		},
		Synonyms:  []string{"relaxed", "leisurely", "unhurried", "listless"},
		Antonyms:  []string{"energetic", "vigorous", "active"},
		ExamLevel: "GRE/Advanced",
	},
	{
		Word:          "taciturn",
		Pronunciation: "/ˈtæs.ɪ.tɜːn/",
		Definitions: []models.Definition{
			{PartOfSpeech: "adjective", Meaning: "Reserved or uncommunicative in speech; saying little.", Example: "He was a quiet, taciturn man who kept his thoughts to himself."},
		},
		Synonyms:  []string{"silent", "reserved", "uncommunicative", "reticent"},
		Antonyms:  []string{"talkative", "loquacious", "garrulous"},
		ExamLevel: "GRE/Advanced",
	},
	{
		Word:          "obsequious",
		Pronunciation: "/əbˈsiː.kwi.əs/",
		Definitions: []models.Definition{
			{PartOfSpeech: "adjective", Meaning: "Obedient or attentive to an excessive or servile degree.", Example: "The waiter bowed with obsequious deference as they entered the restaurant."},
		},
		Synonyms:  []string{"servile", "fawning", "sycophantic", "submissive"},
		Antonyms:  []string{"domineering", "independent", "assertive"},
		ExamLevel: "GRE/Advanced",
	},
}
