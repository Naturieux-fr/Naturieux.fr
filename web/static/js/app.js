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

        // Data
        difficulties: [
            { id: 'beginner', name: 'Debutant', icon: '🌱', desc: '4 choix • 30s' },
            { id: 'intermediate', name: 'Intermediaire', icon: '🌿', desc: '6 choix • 20s' },
            { id: 'expert', name: 'Expert', icon: '🌳', desc: '8 choix • 15s' },
            { id: 'master', name: 'Maitre', icon: '🏔️', desc: '10 choix • 10s' }
        ],

        categories: [
            { id: '', name: 'Toutes', icon: '🌍' },
            { id: 'Mammalia', name: 'Mammiferes', icon: '🦁' },
            { id: 'Aves', name: 'Oiseaux', icon: '🦅' },
            { id: 'Reptilia', name: 'Reptiles', icon: '🦎' },
            { id: 'Amphibia', name: 'Amphibiens', icon: '🐸' },
            { id: 'Insecta', name: 'Insectes', icon: '🦋' },
            { id: 'Plantae', name: 'Plantes', icon: '🌸' },
            { id: 'Fungi', name: 'Champignons', icon: '🍄' }
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

        // Get trophy based on accuracy
        getTrophy() {
            if (this.accuracy >= 90) return '🏆';
            if (this.accuracy >= 70) return '🥈';
            if (this.accuracy >= 50) return '🥉';
            return '🎯';
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
