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

        init() {
            this.token = localStorage.getItem('naturieux_admin_token') || '';
        },

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
