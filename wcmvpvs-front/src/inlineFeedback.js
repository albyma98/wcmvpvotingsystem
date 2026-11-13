// The default survey measures voting ease. Its stored answers retain that meaning.
const experienceScale = [
  { value: 'very_easy', label: 'Facilissima', icon: '🤩', emoji: '🤩' },
  { value: 'easy', label: 'Abbastanza semplice', icon: '🙂', emoji: '😊' },
  { value: 'complex', label: 'Un po’ macchinosa', icon: '😐', emoji: '😐' },
  { value: 'hard', label: 'Difficile', icon: '😣', emoji: '☹️' },
];

export function inlineFeedbackOptions(question) {
  if (question?.id !== 'experience' || !String(question.title || '').trim()
    || !Array.isArray(question.answers) || question.answers.length !== experienceScale.length) return [];
  const answers = experienceScale.map(({ value }) => question.answers.find((answer) => answer?.value === value));
  const labels = answers.map((answer) => String(answer?.label || '').trim());
  const icons = answers.map((answer) => String(answer?.icon || '').trim());
  // A customized scale is safe to display when the club explicitly chose these four icons.
  const explicitScale = experienceScale.every(({ emoji }, index) => icons[index] === emoji);
  const defaultScale = experienceScale.every(({ label, icon }, index) => labels[index] === label && icons[index] === icon);
  if (!labels.every(Boolean) || (!explicitScale && !defaultScale)) return [];
  return experienceScale.map(({ value, emoji }, index) => ({ value, emoji, label: labels[index] }));
}

export function inlineFeedbackQuestions(survey) {
  if (!Array.isArray(survey.questions)) return [];
  const questions = survey.questions.filter((question) =>
    ['experience', 'team_spirit', 'perks_interest', 'mini_games_interest'].includes(question?.id)
    && question.title && Array.isArray(question.answers) && question.answers.length,
  );
  if (!questions.length) return [];
  return [...questions, {
    id: 'report_experience', title: 'Com’è stata la tua esperienza complessiva?',
    answers: [
      { value: 'awesome', label: 'Fantastica' }, { value: 'good', label: 'Bella' },
      { value: 'ok', label: 'Discreta' }, { value: 'bad', label: 'Negativa' },
    ],
  }, { id: 'suggestion', title: survey.suggestionPrompt }];
}
