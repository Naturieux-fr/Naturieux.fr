// Naturieux Quiz App - Alpine.js Component

function quizApp() {
    return {
        // Screens
        screen: 'home', // 'home', 'quiz', 'results', 'leaderboard'
        loading: false,
        error: '',

        // Leaderboard
        leaderboard: [],

        // Dev mode
        devMode: false,

        // Player account (server-side; id kept in localStorage)
        player: {
            id: '',
            name: '',
            level: 1,
            xp: 0,
            xpNext: 100,
            xpPercent: 0,
            totalXp: 0,
            totalGames: 0,
            bestStreak: 0
        },
        usernameInput: '',
        registering: false,

        // Settings
        settings: {
            difficulty: 'beginner',
            taxon: '',
            questionCount: 10
        },

        // Quiz state
        sessionId: '',
        question: {},
        currentQuestion: 0,
        totalQuestions: 10,
        timer: 30,
        timeLimit: 30,
        timerInterval: null,
        startTime: null,
        imageLoaded: false,
        timerPaused: true, // Timer paused until image loads

        // Answer state
        selectedAnswer: null,
        showFeedback: false,
        isCorrect: false,
        correctAnswer: null,
        feedbackText: '',
        lastScore: 0,
        sessionComplete: false,

        // Stats
        score: 0,
        streak: 0,
        bestStreak: 0,
        correctCount: 0,
        accuracy: 0,
        xpGained: 0,

        // Data — difficulty levels are shown as engraved roman numerals
        difficulties: [
            { id: 'beginner', name: 'Apprenti', icon: 'I', desc: '4 choix · 30s' },
            { id: 'intermediate', name: 'Amateur', icon: 'II', desc: '6 choix · 20s' },
            { id: 'expert', name: 'Expert', icon: 'III', desc: '8 choix · 15s' },
            { id: 'master', name: 'Maître', icon: 'IV', desc: '10 choix · 10s' }
        ],

        // Categories use hand-drawn line glyphs (rendered via x-html)
        categories: [
            { id: '', name: 'Toutes', svg: '<svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="8"/><path d="M4 12h16M12 4c2.5 2.4 2.5 13.2 0 16M12 4c-2.5 2.4-2.5 13.2 0 16"/></svg>' },
            { id: 'Mammalia', name: 'Mammifères', svg: '<svg viewBox="0 0 24 24"><circle cx="9" cy="9.5" r="1.4"/><circle cx="15" cy="9.5" r="1.4"/><circle cx="6.5" cy="14" r="1.4"/><circle cx="17.5" cy="14" r="1.4"/><path d="M9 18c0-2.5 6-2.5 6 0 0 1.7-1.4 2.4-3 2.4S9 19.7 9 18Z"/></svg>' },
            { id: 'Aves', name: 'Oiseaux', svg: '<svg viewBox="0 0 24 24"><path d="M4 16c4-1 6-4 6.5-8 .8 3 2.5 5 5.5 5.5-1 2.5-3.5 4.5-7 4.5-2 0-3.5-.7-5-2Z"/><path d="M16 13.5 20 11"/><circle cx="10.5" cy="8.2" r=".6"/></svg>' },
            { id: 'Reptilia', name: 'Reptiles', svg: '<svg viewBox="0 0 24 24"><path d="M4 14c2-3 4 0 6-2s2-5 5-5 4 3 2 5"/><path d="M17 12c2 1 3 3 3 5"/><circle cx="6" cy="6.5" r=".6"/></svg>' },
            { id: 'Amphibia', name: 'Amphibiens', svg: '<svg viewBox="0 0 24 24"><path d="M6 13c0-4 12-4 12 0 0 3-2.5 5-6 5s-6-2-6-5Z"/><circle cx="9" cy="8.5" r="1.6"/><circle cx="15" cy="8.5" r="1.6"/><path d="M5 19l2-2M19 19l-2-2"/></svg>' },
            { id: 'Insecta', name: 'Insectes', svg: '<svg viewBox="0 0 24 24"><path d="M12 5v14M12 9c-3-4-8-2-8 1s5 4 8 1M12 9c3-4 8-2 8 1s-5 4-8 1M12 15c-2.5-2-6-1-6 1.5M12 15c2.5-2 6-1 6 1.5"/></svg>' },
            { id: 'Plantae', name: 'Plantes', svg: '<svg viewBox="0 0 24 24"><path d="M12 21V7"/><path d="M12 13c-3 0-5-2-5-5 3 0 5 2 5 5ZM12 11c2.5 0 4-1.6 4-4-2.5 0-4 1.6-4 4Z"/></svg>' },
            { id: 'Fungi', name: 'Champignons', svg: '<svg viewBox="0 0 24 24"><path d="M4 11c0-4 3.6-6 8-6s8 2 8 6c0 1-7 1.6-8 1.6S4 12 4 11Z"/><path d="M10 12.5c0 4 .5 6-1 8M14 12.5c0 4-.5 6 1 8"/></svg>' }
        ],

        // Initialize
        init() {
            this.loadPlayer();
            this.setTimeLimit();
            this.loadConfig();
        },

        // Load server config
        async loadConfig() {
            try {
                const data = await this.api('/config', 'GET');
                this.devMode = data.dev_mode;
                if (this.devMode) {
                    console.log('🔧 DEV MODE ENABLED');
                }
            } catch (e) {
                console.error('Failed to load config:', e);
            }
        },

        // Called when quiz image loads
        onImageLoad() {
            this.imageLoaded = true;
            if (this.timerPaused && this.screen === 'quiz' && !this.showFeedback) {
                this.timerPaused = false;
                this.startTime = Date.now();
                this.startTimer();
            }
        },

        // Called when quiz image fails to load
        onImageError() {
            console.error('Image failed to load');
            this.imageLoaded = true; // Consider as loaded to not block the quiz
            if (this.timerPaused && this.screen === 'quiz' && !this.showFeedback) {
                this.timerPaused = false;
                this.startTime = Date.now();
                this.startTimer();
            }
        },

        // Load the saved account and refresh the profile from the server
        async loadPlayer() {
            const saved = localStorage.getItem('naturieux_account');
            if (!saved) {
                return;
            }
            try {
                const account = JSON.parse(saved);
                if (account.id) {
                    await this.fetchPlayer(account.id);
                }
            } catch (e) {
                console.error('Error loading account:', e);
            }
        },

        // Fetch the player profile from the server
        async fetchPlayer(id) {
            try {
                const data = await this.api(`/players/${id}`, 'GET');
                this.applyPlayer(data);
            } catch (e) {
                // Unknown account (e.g. wiped database): ask for a new pseudo
                console.error('Fetch player error:', e);
                localStorage.removeItem('naturieux_account');
                this.player.id = '';
            }
        },

        // Map a server player profile onto the local state
        applyPlayer(data) {
            this.player.id = data.id;
            this.player.name = data.username;
            this.player.level = data.level;
            this.player.xp = data.xp_in_level;
            this.player.xpNext = data.xp_for_level;
            this.player.totalXp = data.total_xp;
            this.player.totalGames = data.total_games;
            this.player.bestStreak = data.best_streak;
            this.updateXpPercent();
        },

        // Create the account from the chosen pseudo
        async createAccount() {
            const username = this.usernameInput.trim();
            if (username.length < 2) {
                this.error = 'Le pseudo doit faire au moins 2 caracteres';
                return;
            }

            this.registering = true;
            this.error = '';
            try {
                const data = await this.api('/players', 'POST', { username });
                this.applyPlayer(data);
                localStorage.setItem('naturieux_account', JSON.stringify({
                    id: data.id,
                    username: data.username
                }));
            } catch (e) {
                this.error = e.message === 'username already taken'
                    ? 'Ce pseudo est deja pris'
                    : e.message;
            } finally {
                this.registering = false;
            }
        },

        // Update XP percentage
        updateXpPercent() {
            this.player.xpPercent = (this.player.xp / this.player.xpNext) * 100;
        },

        // Set time limit based on difficulty
        setTimeLimit() {
            const limits = {
                'beginner': 30,
                'intermediate': 20,
                'expert': 15,
                'master': 10
            };
            this.timeLimit = limits[this.settings.difficulty] || 30;
            this.timer = this.timeLimit;
        },

        // API call helper
        async api(endpoint, method = 'GET', body = null) {
            const options = {
                method,
                headers: {
                    'Content-Type': 'application/json'
                }
            };
            if (body) {
                options.body = JSON.stringify(body);
            }

            const response = await fetch(`/api/v1${endpoint}`, options);
            const data = await response.json();

            if (!data.success) {
                throw new Error(data.error || 'Une erreur est survenue');
            }

            return data.data;
        },

        // Start new game
        async startGame() {
            this.loading = true;
            this.error = '';
            this.setTimeLimit();

            try {
                const data = await this.api('/quiz/start', 'POST', {
                    user_id: this.player.id,
                    difficulty: this.settings.difficulty,
                    quiz_types: ['image'],
                    taxon_filter: this.settings.taxon,
                    question_count: this.settings.questionCount
                });

                this.sessionId = data.session_id;
                this.totalQuestions = data.total_questions;
                this.currentQuestion = 1;
                this.score = 0;
                this.streak = 0;
                this.bestStreak = 0;
                this.correctCount = 0;
                this.accuracy = 0;
                this.sessionComplete = false;

                this.loadQuestion(data.question);
                this.screen = 'quiz';
                // Timer will start when image loads (see onImageLoad)

            } catch (e) {
                this.error = e.message;
                console.error('Start game error:', e);
            } finally {
                this.loading = false;
            }
        },

        // Load question
        loadQuestion(q) {
            this.imageLoaded = false;
            this.timerPaused = true; // Pause timer until image loads
            this.question = {
                id: q.id,
                mediaUrl: q.media_url,
                mediaAttribution: q.media_attribution || '',
                choices: q.choices || []
            };
            this.selectedAnswer = null;
            this.showFeedback = false;
            this.timer = this.timeLimit;
            // startTime will be set when image loads
        },

        // Start timer
        startTimer() {
            this.stopTimer();
            this.timerInterval = setInterval(() => {
                this.timer--;
                if (this.timer <= 0) {
                    this.timeOut();
                }
            }, 1000);
        },

        // Stop timer
        stopTimer() {
            if (this.timerInterval) {
                clearInterval(this.timerInterval);
                this.timerInterval = null;
            }
        },

        // Time out - auto submit wrong answer
        timeOut() {
            this.stopTimer();
            if (!this.showFeedback) {
                this.submitAnswer(null);
            }
        },

        // Submit answer
        async submitAnswer(speciesId) {
            if (this.showFeedback) return;

            this.stopTimer();
            this.selectedAnswer = speciesId;
            const timeTaken = Date.now() - this.startTime;

            try {
                const data = await this.api('/quiz/answer', 'POST', {
                    session_id: this.sessionId,
                    species_id: speciesId || 0,
                    time_taken_ms: timeTaken
                });

                this.isCorrect = data.is_correct;
                this.correctAnswer = data.correct_species_id;
                this.feedbackText = `C'etait: ${data.correct_name}`;
                this.lastScore = data.score;
                this.score = data.total_score;
                this.streak = data.current_streak;
                this.accuracy = data.accuracy;
                this.sessionComplete = data.session_complete;

                if (this.isCorrect) {
                    this.correctCount++;
                    if (this.streak > this.bestStreak) {
                        this.bestStreak = this.streak;
                    }
                }

                // Store next question if available
                if (data.next_question) {
                    this.nextQuestionData = data.next_question;
                }

                this.showFeedback = true;

            } catch (e) {
                this.error = e.message;
                console.error('Submit answer error:', e);
            }
        },

        // Next question or results
        nextQuestion() {
            if (this.sessionComplete) {
                this.showResults();
            } else {
                this.currentQuestion++;
                this.loadQuestion(this.nextQuestionData);
                // Timer will start when image loads (see onImageLoad)
            }
        },

        // Show results
        showResults() {
            this.stopTimer();

            // The server is the source of truth for XP: refresh the
            // profile and display the difference
            const previousXp = this.player.totalXp;
            this.xpGained = 0;
            this.fetchPlayer(this.player.id).then(() => {
                this.xpGained = Math.max(0, this.player.totalXp - previousXp);
            });

            this.screen = 'results';
        },

        // Play again with same settings
        playAgain() {
            this.startGame();
        },

        // Go back to home
        goHome() {
            this.stopTimer();
            this.screen = 'home';
        },

        // Show the leaderboard screen
        async showLeaderboard() {
            this.loading = true;
            try {
                const data = await this.api('/leaderboard?limit=10', 'GET');
                this.leaderboard = data.entries || [];
                this.screen = 'leaderboard';
            } catch (e) {
                this.error = 'Impossible de charger le classement';
                console.error('Leaderboard error:', e);
            } finally {
                this.loading = false;
            }
        },

        // Quit current game
        async quitGame() {
            this.stopTimer();

            try {
                await this.api('/quiz/abandon', 'POST', {
                    session_id: this.sessionId
                });
            } catch (e) {
                console.error('Quit game error:', e);
            }

            this.screen = 'home';
        }
    };
}
