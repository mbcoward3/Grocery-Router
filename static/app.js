document.addEventListener('submit', (event) => {
  const form = event.target.closest('[data-wait-form]');
  if (!form) return;
  const button = form.querySelector('button[type="submit"]');
  const status = form.querySelector('.form-status');
  if (button) {
    button.disabled = true;
    button.textContent = button.dataset.waitLabel || 'Working…';
  }
  if (status) status.textContent = 'Keep this page open. The planner usually takes about a minute.';
});

document.addEventListener('change', async (event) => {
  const incoming = event.target.closest('[data-include-form] [name="recipe"]');
  if (incoming) {
    const form = incoming.closest('[data-include-form]');
    const outgoing = form.querySelector('[name="outgoing"]');
    const atUnknownCap = Number(form.dataset.unknownCount) >= 2;
    const incomingIsUnknown = incoming.selectedOptions[0]?.dataset.unknown === 'true';
    for (const option of outgoing.options) {
      option.disabled = atUnknownCap && incomingIsUnknown
        && option.dataset.unknown !== 'true';
    }
    if (outgoing.selectedOptions[0]?.disabled) {
      const allowed = Array.from(outgoing.options).filter(option => !option.disabled);
      if (allowed.length) outgoing.value = allowed[allowed.length - 1].value;
    }
    return;
  }

  const input = event.target.closest('[data-list-key]');
  if (!input) return;
  const row = input.closest('.list-row');
  input.disabled = true;
  try {
    const body = new URLSearchParams({key: input.dataset.listKey});
    const response = await fetch('/api/toggle', {
      method: 'POST',
      headers: {'Content-Type': 'application/x-www-form-urlencoded'},
      body: body.toString()
    });
    if (!response.ok) throw new Error('tick was not saved');
    const result = await response.json();
    input.checked = result.checked;
    row.classList.toggle('is-checked', result.checked);
    const done = document.querySelector('[data-done]');
    if (done) done.textContent = document.querySelectorAll('[data-list-key]:checked').length;
  } catch (error) {
    input.checked = !input.checked;
    row.classList.toggle('is-checked', input.checked);
    window.alert('That tick was not saved. Keep this page open and try again.');
  } finally {
    input.disabled = false;
  }
});
