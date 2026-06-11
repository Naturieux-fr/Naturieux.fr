// Naturieux Quiz App - Alpine.js Component

function quizApp() {
    return {
        // Screens
        screen: 'home', // 'home', 'quiz', 'results', 'leaderboard'
        loading: false,
        error: '',

        // Leaderboard
        leaderboard: [],

        // Multiplayer
        inRoom: false,
        roomDone: false,
        joinCode: '',
        room: { code: '', status: '', players: [], total: 0, round: 0, hostId: '', mode: 'classic', answerMode: 'choices' },
        roomToken: '',
        roomWaiting: false,
        roomSocket: null,
        _wsRetry: null,
        roomMode: 'classic',
        roomModes: [
            { id: 'classic', name: 'Classique', desc: 'le meilleur score gagne' },
            { id: 'elimination', name: 'Élimination', desc: 'une erreur et c\'est fini' }
        ],
        roomPoll: null,
        comboFlash: 0,

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
            bestStreak: 0,
            dailyStreak: 0,
            achievements: []
        },
        usernameInput: '',
        registering: false,

        // Toasts (achievements, level-ups, daily streak)
        toasts: [],
        _toastSeq: 0,
        _preGame: null,
        achievementsCatalogue: [
            { id: 'first_game', icon: '🌱', name: 'Première sortie', desc: 'Terminer une première partie' },
            { id: 'perfect_score', icon: '💯', name: 'Sans-faute', desc: '100 % sur au moins 10 planches' },
            { id: 'streak_master', icon: '⚡', name: 'Série de dix', desc: '10 bonnes réponses d\'affilée' },
            { id: 'expert_mode', icon: '🧭', name: 'Épreuve experte', desc: 'Terminer un quiz en mode Expert' },
            { id: 'master_natural', icon: '👑', name: 'Maître naturaliste', desc: 'Mode Maître réussi à 80 %+' },
            { id: 'level_ten', icon: '🌿', name: 'Naturaliste confirmé', desc: 'Atteindre le niveau 10' },
            { id: 'level_fifty', icon: '🌳', name: 'Sommité du vivant', desc: 'Atteindre le niveau 50' },
            { id: 'dedicated', icon: '🔥', name: 'Assidu', desc: 'Jouer 7 jours d\'affilée' },
            { id: 'veteran', icon: '🎖️', name: 'Vétéran', desc: 'Terminer 100 parties' },
            { id: 'mammal_expert', icon: '🦊', name: 'Mammalogiste', desc: '100 mammifères identifiés' },
            { id: 'bird_watcher', icon: '🦉', name: 'Ornithologue', desc: '100 oiseaux identifiés' },
            { id: 'bug_hunter', icon: '🐝', name: 'Entomologiste', desc: '100 insectes identifiés' },
            { id: 'botanist', icon: '🌸', name: 'Botaniste', desc: '100 plantes identifiées' }
        ],

        // Settings
        settings: {
            difficulty: 'beginner',
            taxon: '',
            questionCount: 10,
            epreuve: 'image',     // image | flash | silhouette | partial
            answerMode: 'choices' // choices | free
        },

        // Quiz-mode runtime state
        quizType: 'image',
        flashHidden: false,
        flashTimeout: null,
        partialStyle: '',
        guess: '',
        attemptsLeft: 3,
        maxAttempts: 3,
        guessHint: '',

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

        epreuves: [
            { id: 'image', name: 'Planche', desc: 'image entière' },
            { id: 'flash', name: 'Éclair', desc: 'un bref instant' },
            { id: 'silhouette', name: 'Silhouette', desc: 'forme sombre' },
            { id: 'partial', name: 'Détail', desc: 'gros plan' }
        ],

        answerModes: [
            { id: 'choices', name: 'Suggestions', desc: 'liste de noms' },
            { id: 'free', name: 'Réponse libre', desc: '3 essais' }
        ],

        // Initialize
        init() {
            this.setTimeLimit();
            this.loadConfig();
            // Resume an in-progress duel after the profile is loaded.
            this.loadPlayer().then(() => this.resumeRoom());
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
            // Flash mode: reveal the image only for a brief moment.
            if (this.quizType === 'flash') {
                this.flashHidden = false;
                this.flashTimeout = setTimeout(() => { this.flashHidden = true; }, this.flashDurationMs);
            }
        },

        // A stable random crop for the "Détail" (partial) mode.
        randomPartialStyle() {
            const tx = Math.round((Math.random() * 50 - 25));
            const ty = Math.round((Math.random() * 50 - 25));
            return `transform: scale(2.7) translate(${tx}%, ${ty}%);`;
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
            this.player.dailyStreak = data.daily_streak || 0;
            this.player.achievements = data.achievements || [];
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

        // Achievement metadata by id (falls back to a generic badge).
        achInfo(id) {
            return this.achievementsCatalogue.find(a => a.id === id) || { id, icon: '🏅', name: id, desc: '' };
        },
        achUnlocked(id) {
            return (this.player.achievements || []).includes(id);
        },
        get achievementsUnlockedCount() {
            return (this.player.achievements || []).length;
        },
        openAchievements() {
            this.error = '';
            this.screen = 'achievements';
        },

        // Show a transient toast (auto-dismissed).
        pushToast(icon, title, text) {
            const id = ++this._toastSeq;
            this.toasts.push({ id, icon, title, text });
            setTimeout(() => { this.toasts = this.toasts.filter(t => t.id !== id); }, 4500);
        },

        // Compare the profile before/after a game and celebrate progress.
        celebrateProgress(before) {
            if (!before) return;
            if (this.player.level > before.level) {
                this.pushToast('✦', 'Niveau supérieur', `Vous voici naturaliste de grade ${this.player.level}`);
            }
            const fresh = (this.player.achievements || []).filter(a => !before.achievements.includes(a));
            for (const a of fresh) {
                const info = this.achInfo(a);
                this.pushToast(info.icon, 'Haut fait débloqué', info.name);
            }
            if (this.player.dailyStreak > before.dailyStreak && this.player.dailyStreak >= 2) {
                this.pushToast('🔥', 'Série quotidienne', `${this.player.dailyStreak} jours d'affilée !`);
            }
        },

        // Start new game
        async startGame() {
            this.loading = true;
            this.error = '';
            this.setTimeLimit();
            this._preGame = { level: this.player.level, achievements: [...(this.player.achievements || [])], dailyStreak: this.player.dailyStreak };

            try {
                const data = await this.api('/quiz/start', 'POST', {
                    user_id: this.player.id,
                    difficulty: this.settings.difficulty,
                    quiz_types: [this.settings.epreuve],
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

        // Load question. The image is fetched as a blob and shown via an
        // object URL, so the underlying file URL is never exposed on the <img>
        // element (no right-click "copy image address", no plain path in the
        // network tab — only the opaque proxy request).
        loadQuestion(q) {
            this.imageLoaded = false;
            this.timerPaused = true; // Pause timer until image loads
            this.question = {
                id: q.id,
                mediaUrl: '',
                mediaAttribution: q.media_attribution || '',
                choices: q.choices || []
            };
            this.selectedAnswer = null;
            this.showFeedback = false;
            this.timer = this.timeLimit;

            // Per-mode setup
            this.quizType = q.quiz_type || 'image';
            this.flashDurationMs = q.flash_duration_ms || 2500;
            this.flashHidden = false;
            if (this.flashTimeout) { clearTimeout(this.flashTimeout); this.flashTimeout = null; }
            this.partialStyle = this.quizType === 'partial' ? this.randomPartialStyle() : '';

            // Free-answer mode
            this.guess = '';
            this.guessHint = '';
            this.attemptsLeft = this.maxAttempts;

            if (this._imageObjectUrl) {
                URL.revokeObjectURL(this._imageObjectUrl);
                this._imageObjectUrl = '';
            }
            if (q.media_url) {
                fetch(q.media_url, { cache: 'no-store' })
                    .then(res => res.ok ? res.blob() : Promise.reject(res.status))
                    .then(blob => {
                        this._imageObjectUrl = URL.createObjectURL(blob);
                        this.question.mediaUrl = this._imageObjectUrl;
                    })
                    .catch(() => this.onImageError());
            }
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

        // Free-answer mode: check a typed guess. Correct → record it; wrong →
        // consume an attempt, and on the last one reveal the answer.
        async submitGuess() {
            if (this.inRoom) { return this.roomGuess(); }
            if (this.showFeedback || !this.guess.trim()) return;
            try {
                const data = await this.api(`/quiz/${this.sessionId}/guess`, 'POST', { guess: this.guess });
                if (data.correct) {
                    this.submitAnswer(data.species_id);
                    return;
                }
                this.attemptsLeft--;
                if (this.attemptsLeft <= 0) {
                    this.submitAnswer(0); // out of attempts: reveal + advance
                } else {
                    this.guessHint = `Pas tout à fait — ${this.attemptsLeft} essai${this.attemptsLeft > 1 ? 's' : ''} restant${this.attemptsLeft > 1 ? 's' : ''}`;
                    this.guess = '';
                }
            } catch (e) {
                this.error = e.message;
            }
        },

        // Submit answer (routes to the room in multiplayer)
        async submitAnswer(speciesId) {
            if (this.inRoom) { return this.roomAnswer(speciesId); }
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
            this.comboFlash = 0;
            if (this.inRoom) {
                // In a duel the game advances automatically once everyone has
                // answered; the only manual step is leaving to the podium once
                // this player is out (eliminated) or the game is over.
                if (this.roomDone) {
                    this.showFeedback = false;
                    this.screen = 'mp-podium';
                }
                return;
            }
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
            // profile and display the difference, then celebrate any progress.
            const previousXp = this.player.totalXp;
            const before = this._preGame;
            this.xpGained = 0;
            this.fetchPlayer(this.player.id).then(() => {
                this.xpGained = Math.max(0, this.player.totalXp - previousXp);
                this.celebrateProgress(before);
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

        // ---------- Multijoueur ----------

        openMulti() {
            this.error = '';
            this.joinCode = '';
            this.roomToken = '';
            this.inRoom = false;
            this.roomDone = false;
            this.roomWaiting = false;
            this.room = { code: '', status: '', players: [], total: 0, round: 0, hostId: '', mode: 'classic', answerMode: 'choices' };
            this.screen = 'mp-lobby';
        },

        async createRoom() {
            this.loading = true; this.error = '';
            try {
                const data = await this.api('/rooms', 'POST', {
                    host_id: this.player.id, host_name: this.player.name,
                    difficulty: this.settings.difficulty, quiz_type: this.settings.epreuve,
                    answer_mode: this.settings.answerMode, mode: this.roomMode,
                    taxon_filter: this.settings.taxon, count: this.settings.questionCount
                });
                this.room.code = data.code;
                this.roomToken = data.token;
                this.startRoomPolling();
            } catch (e) { this.error = e.message; }
            finally { this.loading = false; }
        },

        async joinRoom() {
            const code = this.joinCode.trim().toUpperCase();
            if (code.length < 4) { this.error = 'Code à 4 lettres'; return; }
            this.loading = true; this.error = '';
            try {
                const data = await this.api(`/rooms/${code}/join`, 'POST', {
                    player_id: this.player.id, player_name: this.player.name
                });
                this.room.code = code;
                this.roomToken = data.token;
                this.applyRoom(data.room);
                this.startRoomPolling();
            } catch (e) {
                this.error = e.message === 'room not found' ? 'Salon introuvable' : e.message;
            } finally { this.loading = false; }
        },

        async startRoom() {
            try {
                await this.api(`/rooms/${this.room.code}/start`, 'POST', { player_id: this.player.id, token: this.roomToken });
            } catch (e) { this.error = e.message; }
        },

        // Live sync: a WebSocket pushes state instantly; a slow poll is the
        // fallback that also drives reconnection.
        startRoomPolling() {
            this.stopRoomPolling();
            this.connectRoomSocket();
            this.roomPoll = setInterval(() => this.pollRoom(), 4000);
            this.pollRoom();
            this.saveRoomSession();
        },
        stopRoomPolling() {
            if (this.roomPoll) { clearInterval(this.roomPoll); this.roomPoll = null; }
            this.closeRoomSocket();
        },

        connectRoomSocket() {
            if (!this.room.code) return;
            this.closeRoomSocket();
            const proto = location.protocol === 'https:' ? 'wss' : 'ws';
            try {
                const ws = new WebSocket(`${proto}://${location.host}/api/v1/rooms/${this.room.code}/ws`);
                this.roomSocket = ws;
                ws.onmessage = (ev) => { try { this.applyRoomState(JSON.parse(ev.data)); } catch (e) {} };
                ws.onerror = () => { try { ws.close(); } catch (e) {} };
                ws.onclose = () => {
                    if (this.roomSocket !== ws) return;
                    this.roomSocket = null;
                    // Auto-reconnect while the room is still live.
                    if (this.room.code && this.room.status !== 'finished') {
                        this._wsRetry = setTimeout(() => this.connectRoomSocket(), 2000);
                    }
                };
            } catch (e) { /* fall back to polling */ }
        },
        closeRoomSocket() {
            if (this._wsRetry) { clearTimeout(this._wsRetry); this._wsRetry = null; }
            if (this.roomSocket) {
                try { this.roomSocket.onclose = null; this.roomSocket.close(); } catch (e) {}
                this.roomSocket = null;
            }
        },

        async pollRoom() {
            if (!this.room.code) return;
            try {
                const data = await this.api(`/rooms/${this.room.code}?player_id=${this.player.id}`, 'GET');
                this.applyRoomState(data.room, data.your_question);
            } catch (e) { /* transient: keep the socket / next poll */ }
        },

        // Unified handler for a room snapshot from either transport.
        applyRoomState(state, yourQuestion) {
            if (!state) return;
            const wasStatus = this.room.status;
            const prevRound = this.room.round || 0;
            this.applyRoom(state);

            if (this.room.status === 'playing') {
                if (!this.inRoom) {
                    // The game just started for me.
                    if (yourQuestion) this.beginRoomQuiz(yourQuestion);
                    else this.fetchMyQuestion();
                } else if (this.room.round > prevRound && !this.roomDone) {
                    // Everyone answered: advance to the next shared question.
                    if (yourQuestion) this.loadRoomRound(yourQuestion);
                    else this.fetchMyQuestion();
                }
            }
            if (this.room.status === 'finished') {
                this.stopRoomPolling();
                this.clearRoomSession();
                if (this.screen !== 'mp-podium') this.screen = 'mp-podium';
            }
        },

        async fetchMyQuestion() {
            try {
                const d = await this.api(`/rooms/${this.room.code}?player_id=${this.player.id}`, 'GET');
                if (!d.your_question) return;
                if (!this.inRoom) this.beginRoomQuiz(d.your_question);
                else this.loadRoomRound(d.your_question);
            } catch (e) {}
        },

        // Load the next shared round's question (clears the feedback/waiting).
        loadRoomRound(question) {
            this.showFeedback = false;
            this.roomWaiting = false;
            this.comboFlash = 0;
            this.currentQuestion = (this.room.round || 0) + 1;
            this.loadQuestion(question);
        },

        // Persist the active room so a reload can resume it.
        saveRoomSession() {
            if (this.room.code && this.roomToken) {
                localStorage.setItem('naturieux_room', JSON.stringify({ code: this.room.code, token: this.roomToken }));
            }
        },
        clearRoomSession() { localStorage.removeItem('naturieux_room'); },

        async resumeRoom() {
            const saved = localStorage.getItem('naturieux_room');
            if (!saved || !this.player.id) return;
            let s;
            try { s = JSON.parse(saved); } catch (e) { this.clearRoomSession(); return; }
            if (!s.code || !s.token) { this.clearRoomSession(); return; }
            try {
                const data = await this.api(`/rooms/${s.code}?player_id=${this.player.id}`, 'GET');
                const inIt = (data.room.players || []).some(p => p.id === this.player.id);
                if (!inIt || data.room.status === 'finished') { this.clearRoomSession(); return; }
                this.room.code = s.code;
                this.roomToken = s.token;
                this.applyRoom(data.room);
                if (data.room.status === 'lobby') {
                    this.screen = 'mp-lobby';
                } else if (data.room.status === 'playing') {
                    if (data.your_question) this.beginRoomQuiz(data.your_question);
                    else this.screen = 'mp-podium'; // already finished my questions; awaiting others
                }
                this.startRoomPolling();
            } catch (e) { this.clearRoomSession(); }
        },

        applyRoom(r) {
            if (!r) return;
            this.room.status = r.status;
            this.room.players = r.players || [];
            this.room.total = r.total_questions || 0;
            this.room.round = r.round || 0;
            this.room.hostId = r.host_id;
            if (r.settings) {
                this.room.mode = r.settings.mode || 'classic';
                this.room.answerMode = r.settings.answer_mode || 'choices';
            }
        },

        // The answer UI follows the room's mode in multiplayer, the local
        // setting in solo.
        get activeAnswerMode() {
            return this.inRoom ? this.room.answerMode : this.settings.answerMode;
        },

        get isHost() { return this.room.hostId === this.player.id; },
        get myStanding() { return this.room.players.find(p => p.id === this.player.id) || {}; },

        beginRoomQuiz(question) {
            this.inRoom = true;
            this.roomDone = false;
            this.roomWaiting = false;
            this.totalQuestions = this.room.total;
            this.currentQuestion = (this.room.round || 0) + 1;
            this.score = 0; this.streak = 0; this.bestStreak = 0; this.correctCount = 0; this.accuracy = 0;
            this.sessionComplete = false;
            this.setTimeLimit();
            this.loadQuestion(question);
            this.screen = 'quiz';
        },

        // Free-answer in a room: check the guess, then record via roomAnswer.
        async roomGuess() {
            if (this.showFeedback || !this.guess.trim()) return;
            try {
                const data = await this.api(`/rooms/${this.room.code}/guess`, 'POST', { token: this.roomToken, guess: this.guess });
                if (data.correct) { this.roomAnswer(data.species_id); return; }
                this.attemptsLeft--;
                if (this.attemptsLeft <= 0) {
                    this.roomAnswer(0);
                } else {
                    this.guessHint = `Pas tout à fait — ${this.attemptsLeft} essai${this.attemptsLeft > 1 ? 's' : ''} restant${this.attemptsLeft > 1 ? 's' : ''}`;
                    this.guess = '';
                }
            } catch (e) { this.error = e.message; }
        },

        async roomAnswer(speciesId) {
            if (this.showFeedback) return;
            this.stopTimer();
            this.selectedAnswer = speciesId;
            try {
                const data = await this.api(`/rooms/${this.room.code}/answer`, 'POST', {
                    player_id: this.player.id, token: this.roomToken, species_id: speciesId || 0
                });
                this.isCorrect = data.is_correct;
                this.correctAnswer = data.correct_species_id;
                this.feedbackText = `C'etait: ${data.correct_name}`;
                this.lastScore = data.score;
                this.score = data.total_score;
                this.streak = data.streak;
                if (this.isCorrect) {
                    this.correctCount++;
                    if (this.streak > this.bestStreak) this.bestStreak = this.streak;
                    if (this.streak >= 3) this.comboFlash = this.streak;
                }
                this.roomDone = data.done;
                this.roomWaiting = !!data.waiting;
                this.sessionComplete = data.done;
                this.showFeedback = true;
            } catch (e) { this.error = e.message; }
        },

        leaveRoom() {
            this.stopRoomPolling();
            this.clearRoomSession();
            this.inRoom = false;
            this.roomDone = false;
            this.roomWaiting = false;
            this.showFeedback = false;
            this.roomToken = '';
            this.room = { code: '', status: '', players: [], total: 0, round: 0, hostId: '', mode: 'classic', answerMode: 'choices' };
            this.goHome();
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
