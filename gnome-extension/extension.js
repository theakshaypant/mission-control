// Mission Control — GNOME Shell extension.
// Requires the mission-control API server to be running (cmd/server).
//
// Install:  make install-gnome-ext
// Update:   make update-gnome-ext

import GObject from 'gi://GObject';
import GLib from 'gi://GLib';
import Gio from 'gi://Gio';
import Soup from 'gi://Soup';
import St from 'gi://St';
import Clutter from 'gi://Clutter';

import * as Main from 'resource:///org/gnome/shell/ui/main.js';
import * as PanelMenu from 'resource:///org/gnome/shell/ui/panelMenu.js';
import * as PopupMenu from 'resource:///org/gnome/shell/ui/popupMenu.js';
import { Extension } from 'resource:///org/gnome/shell/extensions/extension.js';

const REFRESH_S = 30;

function truncate(s, n) {
    return s.length <= n ? s : s.slice(0, n - 1) + '…';
}

const Indicator = GObject.registerClass(
    class Indicator extends PanelMenu.Button {
        _init(apiUrl) {
            super._init(0.0, 'Mission Control');

            this._api         = apiUrl.replace(/\/$/, '');
            this._session     = new Soup.Session();
            this._cancellable = new Gio.Cancellable();
            this._refreshing  = false;
            this._destroyed   = false;

            this._label = new St.Label({
                text:    'MC',
                y_align: Clutter.ActorAlign.CENTER,
                style:   'font-family: monospace; font-size: 11px; padding: 0 4px;',
            });
            this.add_child(this._label);

            const placeholder = new PopupMenu.PopupMenuItem('Loading…', { reactive: false });
            placeholder.label.set_style('color: #6b7280;');
            this.menu.addMenuItem(placeholder);

            this.menu.connect('open-state-changed', (_menu, open) => {
                if (open) this._refresh();
            });

            this._refresh();
            this._timer = GLib.timeout_add_seconds(
                GLib.PRIORITY_DEFAULT,
                REFRESH_S,
                () => { this._refresh(); return GLib.SOURCE_CONTINUE; },
            );
        }

        async _get(path) {
            const msg   = Soup.Message.new('GET', this._api + path);
            const bytes = await this._session.send_and_read_async(
                msg, GLib.PRIORITY_DEFAULT, this._cancellable,
            );
            if (msg.get_status() !== Soup.Status.OK)
                throw new Error(`HTTP ${msg.get_status()}`);
            return JSON.parse(new TextDecoder().decode(bytes.get_data()));
        }

        async _post(path) {
            const msg = Soup.Message.new('POST', this._api + path);
            await this._session.send_and_read_async(
                msg, GLib.PRIORITY_DEFAULT, this._cancellable,
            );
            if (msg.get_status() !== Soup.Status.OK)
                throw new Error(`HTTP ${msg.get_status()}`);
        }

        async _refresh() {
            if (this._refreshing) return;
            this._refreshing = true;
            try {
                const priority = await this._get('/summary');
                if (this._destroyed) return;
                const count = priority.length;
                this._label.set_text(count > 0 ? `MC · ${count}` : 'MC');
                this._buildMenu(priority);
            } catch (_e) {
                if (this._destroyed) return;
                this._label.set_text('MC ⚠');
                this._buildErrorMenu();
            } finally {
                this._refreshing = false;
            }
        }

        _addStatic(text, style) {
            const item = new PopupMenu.PopupMenuItem(text, { reactive: false });
            if (style) item.label.set_style(style);
            this.menu.addMenuItem(item);
            return item;
        }

        _buildMenu(priority) {
            this.menu.removeAll();

            this._addStatic(
                `MISSION CONTROL  ·  Needs attention: ${priority.length}`,
                'font-family: monospace; font-size: 10px; color: #9ca3af;',
            );
            this.menu.addMenuItem(new PopupMenu.PopupSeparatorMenuItem());

            if (priority.length === 0) {
                this._addStatic('Nothing needs your attention.', 'color: #6b7280;');
            }

            for (const item of priority) {
                const row = new PopupMenu.PopupMenuItem(
                    `${item.type.toUpperCase()}  ${truncate(item.title, 60)}`,
                );
                row.label.set_style('font-family: monospace; font-size: 11px;');
                row.connect('activate', () => {
                    try { Gio.AppInfo.launch_default_for_uri(item.url, null); } catch (_e) {}
                });
                this.menu.addMenuItem(row);

                const parts = [];
                if (item.namespace)              parts.push(item.namespace);
                if (item.active_signals?.length) parts.push(item.active_signals.join(', '));
                if (parts.length > 0) {
                    this._addStatic(
                        `  ${parts.join('  ·  ')}`,
                        'font-size: 10px; color: #6b7280;',
                    );
                }
            }

            this.menu.addMenuItem(new PopupMenu.PopupSeparatorMenuItem());

            const dash = new PopupMenu.PopupMenuItem('Open Dashboard');
            dash.connect('activate', () => {
                try { Gio.AppInfo.launch_default_for_uri(this._api, null); } catch (_e) {}
            });
            this.menu.addMenuItem(dash);

            const sync = new PopupMenu.PopupMenuItem('Sync Now');
            sync.connect('activate', () =>
                this._post('/sync').then(() => this._refresh()).catch(() => {}),
            );
            this.menu.addMenuItem(sync);
        }

        _buildErrorMenu() {
            this.menu.removeAll();
            this._addStatic('Could not reach the API server.', 'color: #ef4444;');
            this._addStatic('  Start it with: mc-server', 'font-family: monospace; font-size: 10px; color: #6b7280;');
        }

        destroy() {
            this._destroyed = true;
            this._cancellable.cancel();
            this._session.abort();
            this._session = null;
            if (this._timer) {
                GLib.source_remove(this._timer);
                this._timer = null;
            }
            super.destroy();
        }
    },
);

export default class MissionControlExtension extends Extension {
    enable() {
        this._settings = this.getSettings('org.gnome.shell.extensions.mission-control');
        const apiUrl = this._settings.get_string('api-url');
        this._indicator = new Indicator(apiUrl);
        Main.panel.addToStatusArea(this.uuid, this._indicator);
    }

    disable() {
        this._indicator?.destroy();
        this._indicator = null;
        this._settings = null;
    }
}
