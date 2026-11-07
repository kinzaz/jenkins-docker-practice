import React, { useEffect, useState } from 'react';
import { createRoot } from 'react-dom/client';
import './style.css';

async function request(path = '', options = {}) {
  const response = await fetch(`/api/todos${path}`, options);
  if (!response.ok) throw new Error('Не удалось выполнить запрос. Проверьте, запущен ли сервер.');
  return response.status === 204 ? null : response.json();
}

function App() {
  const [todos, setTodos] = useState([]);
  const [title, setTitle] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);

  async function load() { setTodos(await request()); }
  useEffect(() => { load().catch(e => setError(e.message)).finally(() => setLoading(false)); }, []);

  async function change(action) {
    setBusy(true);
    setError('');
    try { await action(); await load(); }
    catch (e) { setError(e.message); }
    finally { setBusy(false); }
  }

  function add(event) {
    event.preventDefault();
    if (!title.trim()) return;
    change(async () => {
      await request('', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ title: title.trim() }) });
      setTitle('');
    });
  }

  return <main>
    <p className="eyebrow">ОДИН ШАГ ЗА РАЗ</p>
    <h1>Мои задачи<span>.</span></h1>
    <p className="subtitle">Освободи голову. Запиши то, что нужно сделать.</p>
    <form onSubmit={add}>
      <input aria-label="Новая задача" placeholder="Что нужно сделать?" value={title} onChange={e => setTitle(e.target.value)} maxLength={200} required disabled={busy || loading} />
      <button disabled={busy || loading || !title.trim()}>Добавить</button>
    </form>
    {error && <p role="alert" className="error">{error} <button disabled={busy} onClick={() => change(async () => {})}>Повторить</button></p>}
    <div className="list" aria-live="polite">
      {loading ? <p className="empty">Загрузка…</p> : !todos.length && !error ? <p className="empty">Пока задач нет. Добавь первую выше.</p> : todos.map(todo => <div className="todo" key={todo.id}>
        <label><input type="checkbox" checked={todo.done} disabled={busy} onChange={() => change(() => request(`/${todo.id}`, { method: 'PATCH', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ done: !todo.done }) }))} /><span className={todo.done ? 'done' : ''}>{todo.title}</span></label>
        <button className="delete" aria-label={`Удалить задачу: ${todo.title}`} disabled={busy} onClick={() => change(() => request(`/${todo.id}`, { method: 'DELETE' }))}>×</button>
      </div>)}
    </div>
    <footer>{todos.filter(t => !t.done).length} осталось · {todos.filter(t => t.done).length} выполнено <span>lesson-7.1</span></footer>
  </main>;
}

createRoot(document.getElementById('root')).render(<App />);
