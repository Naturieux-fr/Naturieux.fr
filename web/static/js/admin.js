// Naturieux admin back-office — photo management (Alpine.js component).

function adminApp() {
    return {
        token: '',
        busy: false,
        error: '',

        login: { username: '', password: '' },
        search: { query: '', results: [] },
        selected: null,
        photos: [],
        form: { mode: 'upload', url: '', file: null, preview: '', attribution: '', license: '', difficulty: '' },
        inviteLink: '',
        tab: 'dashboard',
        stats: {},
        players: [],
        coverage: [],
        invites: [],
        quizzes: [],
        newQuiz: { name: '', species: [] },
        quizSearch: { query: '', results: [] },

        init() {
            this.token = localStorage.getItem('naturieux_admin_token') || '';
            if (this.token) this.loadStats();
        },

        // Dashboard
        async loadStats() {
            try {
                this.stats = await this.api('/admin/stats', 'GET');
                try { this.coverage = (await this.api('/admin/coverage', 'GET')).coverage || []; } catch (e) { this.coverage = []; }
            } catch (e) { this.error = e.message; }
        },

        // Players
        async loadPlayers() {
            try { this.players = (await this.api('/admin/players', 'GET')).players || []; }
            catch (e) { this.error = e.message; }
        },
        async setRole(id, role) {
            try { await this.api(`/admin/players/${id}/role`, 'POST', { role }); await this.loadPlayers(); }
            catch (e) { this.error = e.message; }
        },
        async deletePlayer(id, name) {
            if (!confirm(`Supprimer le compte « ${name} » ?`)) return;
            try { await this.api(`/admin/players/${id}`, 'DELETE'); await this.loadPlayers(); await this.loadStats(); }
            catch (e) { this.error = e.message; }
        },

        // ----- Curated quizzes & challenge scheduling -----
        async loadQuizzes() {
            try { this.quizzes = (await this.api('/admin/quizzes', 'GET')).quizzes || []; }
            catch (e) { this.error = e.message; }
        },
        async searchForQuiz() {
            if (!this.quizSearch.query.trim()) return;
            try { this.quizSearch.results = (await this.api(`/admin/taxa?q=${encodeURIComponent(this.quizSearch.query)}`, 'GET')).taxa || []; }
            catch (e) { this.error = e.message; }
        },
        addToQuiz(t) {
            if (!this.newQuiz.species.some(s => s.cd_nom === t.cd_nom)) {
                this.newQuiz.species.push({ cd_nom: t.cd_nom, scientific_name: t.scientific_name });
            }
        },
        removeFromQuiz(cd) { this.newQuiz.species = this.newQuiz.species.filter(s => s.cd_nom !== cd); },
        async createQuiz() {
            if (!this.newQuiz.name.trim() || !this.newQuiz.species.length) { this.error = 'Nom et au moins une espèce requis'; return; }
            try {
                await this.api('/admin/quizzes', 'POST', { name: this.newQuiz.name, species: this.newQuiz.species.map(s => s.cd_nom) });
                this.newQuiz = { name: '', species: [] };
                this.quizSearch = { query: '', results: [] };
                await this.loadQuizzes();
            } catch (e) { this.error = e.message; }
        },
        async deleteQuiz(id, name) {
            if (!confirm(`Supprimer le quiz « ${name} » ?`)) return;
            try { await this.api(`/admin/quizzes/${id}`, 'DELETE'); await this.loadQuizzes(); }
            catch (e) { this.error = e.message; }
        },
        async scheduleChallenge(quizId, period) {
            try {
                const d = await this.api('/admin/challenge/schedule', 'POST', { period, quiz_id: quizId });
                this.error = '';
                alert(`Programmé comme ${period === 'weekly' ? 'défi de la semaine' : 'défi du jour'} (${d.key}).`);
            } catch (e) { this.error = e.message; }
        },

        // Generate a player invitation link to share.
        async generateInvite() {
            this.error = '';
            try {
                const data = await this.api('/admin/invites', 'POST', {});
                this.inviteLink = `${location.origin}/?invite=${encodeURIComponent(data.invite)}`;
                await this.loadInvites();
            } catch (e) { this.error = e.message; }
        },
        async copyInvite() {
            try { await navigator.clipboard.writeText(this.inviteLink); } catch (e) {}
        },
        async loadInvites() {
            try { this.invites = (await this.api('/admin/invites', 'GET')).invites || []; }
            catch (e) { this.error = e.message; }
        },
        async revokeInvite(token) {
            try { await this.api(`/admin/invites/${token}/revoke`, 'POST', {}); await this.loadInvites(); }
            catch (e) { this.error = e.message; }
        },
        inviteStatus(i) {
            if (i.revoked) return 'révoquée';
            if (i.used_by) return 'utilisée';
            return 'en attente';
        },
        inviteLinkFor(token) { return `${location.origin}/?invite=${encodeURIComponent(token)}`; },
        async copyText(t) { try { await navigator.clipboard.writeText(t); } catch (e) {} },

        // Authenticated API call. Returns the `data` payload or throws.
        async api(endpoint, method = 'GET', body = null) {
            const options = { method, headers: {} };
            if (this.token) {
                options.headers['Authorization'] = 'Bearer ' + this.token;
            }
            if (body) {
                options.headers['Content-Type'] = 'application/json';
                options.body = JSON.stringify(body);
            }
            const res = await fetch('/api/v1' + endpoint, options);
            if (res.status === 401 || res.status === 403) {
                this.logout();
                throw new Error('Session expirée, reconnectez-vous');
            }
            const data = await res.json();
            if (!data.success) {
                throw new Error(data.error || 'Erreur');
            }
            return data.data;
        },

        async doLogin() {
            this.busy = true;
            this.error = '';
            try {
                const data = await this.api('/auth/login', 'POST', {
                    username: this.login.username,
                    password: this.login.password
                });
                this.token = data.token;
                localStorage.setItem('naturieux_admin_token', this.token);
                this.login.password = '';
                this.loadStats();
            } catch (e) {
                this.error = e.message;
            } finally {
                this.busy = false;
            }
        },

        logout() {
            this.token = '';
            localStorage.removeItem('naturieux_admin_token');
            this.selected = null;
            this.photos = [];
        },

        async doSearch() {
            if (!this.search.query.trim()) return;
            this.busy = true;
            this.error = '';
            try {
                const data = await this.api('/admin/taxa?q=' + encodeURIComponent(this.search.query));
                this.search.results = data.taxa || [];
            } catch (e) {
                this.error = e.message;
            } finally {
                this.busy = false;
            }
        },

        async select(taxon) {
            this.selected = taxon;
            this.error = '';
            await this.loadPhotos();
        },

        async loadPhotos() {
            if (!this.selected) return;
            try {
                const data = await this.api('/admin/taxa/' + this.selected.cd_nom + '/photos');
                this.photos = data.photos || [];
            } catch (e) {
                this.error = e.message;
            }
        },

        // ----- Zones d'image (zoom Détail + espèces multiples) -----
        zoneEditor: { open: false, photoId: null, url: '', zoom: null, species: [], tool: 'zoom', toolSpecies: null, drawing: null },
        zoneSearch: { query: '', results: [] },
        _zStart: null,

        openZones(p) {
            let z = {};
            try { z = typeof p.zones === 'string' ? JSON.parse(p.zones || '{}') : (p.zones || {}); } catch (e) { z = {}; }
            this.zoneEditor = {
                open: true, photoId: p.id, url: p.url,
                zoom: z.zoom || null, species: z.species || [],
                tool: 'zoom', toolSpecies: null, drawing: null
            };
            this.zoneSearch = { query: '', results: [] };
        },
        closeZones() { this.zoneEditor.open = false; this._zStart = null; },
        selectZoomTool() { this.zoneEditor.tool = 'zoom'; this.zoneEditor.toolSpecies = null; },
        async searchZoneSpecies() {
            if (!this.zoneSearch.query.trim()) return;
            try { this.zoneSearch.results = (await this.api(`/admin/taxa?q=${encodeURIComponent(this.zoneSearch.query)}`, 'GET')).taxa || []; }
            catch (e) { this.error = e.message; }
        },
        pickZoneSpecies(t) { this.zoneEditor.tool = 'species'; this.zoneEditor.toolSpecies = { cd_nom: t.cd_nom, name: t.scientific_name }; },
        clearZoom() { this.zoneEditor.zoom = null; },
        removeSpeciesZone(i) { this.zoneEditor.species.splice(i, 1); },
        _frac(e) {
            const r = e.currentTarget.getBoundingClientRect();
            return { x: Math.min(1, Math.max(0, (e.clientX - r.left) / r.width)), y: Math.min(1, Math.max(0, (e.clientY - r.top) / r.height)) };
        },
        zoneDown(e) { const p = this._frac(e); this._zStart = p; this.zoneEditor.drawing = { x: p.x, y: p.y, w: 0, h: 0 }; },
        zoneMove(e) {
            if (!this._zStart) return;
            const p = this._frac(e), s = this._zStart;
            this.zoneEditor.drawing = { x: Math.min(s.x, p.x), y: Math.min(s.y, p.y), w: Math.abs(p.x - s.x), h: Math.abs(p.y - s.y) };
        },
        zoneUp() {
            const d = this.zoneEditor.drawing; this._zStart = null; this.zoneEditor.drawing = null;
            if (!d || d.w < 0.02 || d.h < 0.02) return;
            const rect = { x: +d.x.toFixed(3), y: +d.y.toFixed(3), w: +d.w.toFixed(3), h: +d.h.toFixed(3) };
            if (this.zoneEditor.tool === 'zoom') this.zoneEditor.zoom = rect;
            else if (this.zoneEditor.toolSpecies) this.zoneEditor.species.push({ ...this.zoneEditor.toolSpecies, ...rect });
        },
        async saveZones() {
            const z = this.zoneEditor;
            try {
                await this.api(`/admin/photos/${z.photoId}/zones`, 'POST', { zoom: z.zoom, species: z.species });
                this.closeZones();
                await this.loadPhotos();
            } catch (e) { this.error = e.message; }
        },

        // Capture the chosen file and show a local preview.
        onFile(event) {
            const file = event.target.files[0];
            this.form.file = file || null;
            this.form.preview = file ? URL.createObjectURL(file) : '';
        },

        resetForm() {
            this.form = { mode: this.form.mode, url: '', file: null, preview: '', attribution: '', license: '', difficulty: '' };
        },

        // Add a photo either by uploading a file or by referencing a URL.
        submitPhoto() {
            return this.form.mode === 'upload' ? this.uploadPhoto() : this.addPhotoByURL();
        },

        async uploadPhoto() {
            if (!this.selected || !this.form.file) {
                this.error = 'Choisissez un fichier image';
                return;
            }
            this.busy = true;
            this.error = '';
            try {
                const fd = new FormData();
                fd.append('file', this.form.file);
                fd.append('attribution', this.form.attribution);
                fd.append('license', this.form.license);
                fd.append('difficulty', this.form.difficulty);

                const res = await fetch('/api/v1/admin/taxa/' + this.selected.cd_nom + '/upload', {
                    method: 'POST',
                    headers: { 'Authorization': 'Bearer ' + this.token },
                    body: fd
                });
                if (res.status === 401 || res.status === 403) {
                    this.logout();
                    throw new Error('Session expirée, reconnectez-vous');
                }
                const data = await res.json();
                if (!data.success) throw new Error(data.error || 'Erreur');
                this.resetForm();
                await this.loadPhotos();
            } catch (e) {
                this.error = e.message;
            } finally {
                this.busy = false;
            }
        },

        async addPhotoByURL() {
            if (!this.selected || !this.form.url.trim()) {
                this.error = 'URL requise';
                return;
            }
            this.busy = true;
            this.error = '';
            try {
                await this.api('/admin/taxa/' + this.selected.cd_nom + '/photos', 'POST', {
                    url: this.form.url,
                    attribution: this.form.attribution,
                    license: this.form.license,
                    difficulty: this.form.difficulty
                });
                this.resetForm();
                await this.loadPhotos();
            } catch (e) {
                this.error = e.message;
            } finally {
                this.busy = false;
            }
        },

        async removePhoto(id) {
            this.busy = true;
            this.error = '';
            try {
                await this.api('/admin/photos/' + id, 'DELETE');
                await this.loadPhotos();
            } catch (e) {
                this.error = e.message;
            } finally {
                this.busy = false;
            }
        }
    };
}
