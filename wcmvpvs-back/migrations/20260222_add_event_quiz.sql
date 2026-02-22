CREATE TABLE IF NOT EXISTS event_quiz (
  event_id INTEGER PRIMARY KEY,
  enabled INTEGER NOT NULL DEFAULT 0,
  questions_per_session INTEGER NOT NULL DEFAULT 5,
  seconds_per_question INTEGER NOT NULL DEFAULT 8,
  base_reward INTEGER NOT NULL DEFAULT 3,
  completion_bonus INTEGER NOT NULL DEFAULT 5,
  streak_bonus INTEGER NOT NULL DEFAULT 1,
  active_from TEXT,
  active_to TEXT,
  FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS event_quiz_questions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  quiz_id INTEGER NOT NULL,
  question_text TEXT NOT NULL,
  answers_json TEXT NOT NULL,
  correct_index INTEGER NOT NULL,
  order_index INTEGER NOT NULL DEFAULT 0,
  FOREIGN KEY (quiz_id) REFERENCES event_quiz(event_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_event_quiz_questions_quiz ON event_quiz_questions(quiz_id, order_index);
