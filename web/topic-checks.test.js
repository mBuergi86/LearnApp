const test = require('node:test');
const assert = require('node:assert/strict');
const {readFileSync} = require('node:fs');
const {join} = require('node:path');
const vm = require('node:vm');
const {topicNavigation} = require('./ui-state.js');
const script = readFileSync(join(__dirname, 'app.js'), 'utf8');
const start = script.indexOf('async function loadTopicChecks(');
const end = script.indexOf('function todaySetupView(', start);
const loader = script.slice(start, end);
const escapeHTML = value => String(value).replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));

function sandbox(requestJSON) {
  const context = vm.createContext({requestJSON, escapeHTML, optionText:(question,id)=>question.options.find(o=>o.id===id)?.text});
  vm.runInContext(loader, context);
  return context;
}
function container(connected=true) {
  return {isConnected:connected,innerHTML:'loading',querySelectorAll:()=>[],querySelector:()=>({addEventListener(){}})};
}
function publicQuestion(topicID='current') {
  return {id:'question',topic_id:topicID,type:'true-false',prompt:'<img src=x onerror=alert(1)>',options:[{id:'true',text:'Richtig'},{id:'false',text:'Falsch'}]};
}

test('topic checks stay on the learning page and module training is last-only', () => {
  const topic = script.slice(script.indexOf('function topicView('),start);
  assert.ok(topic.includes('const moduleTraining = !sequence.nextID ?'));
  assert.ok(topic.includes("${!sequence.nextID ? `<a data-topic-training"));
  assert.ok(topic.includes('Kurzcheck: dieses Thema'));
  assert.ok(topic.includes("loadTopicChecks(topic.id, document.querySelector('.topic-check-list'))"));
  assert.ok(topic.indexOf('${moduleTraining}') > topic.indexOf('topic-check-list'));
  const topics = [{id:'a',module_id:'m',exam_id:'ml'},{id:'b',module_id:'m',exam_id:'ml'},{id:'c',module_id:'m',exam_id:'ml'}];
  assert.deepEqual(topics.map(t=>!topicNavigation(topics,t.id).nextID),[false,false,true]);
  assert.ok(!loader.includes("'/api/answers'"));
  assert.ok(loader.includes("'/api/comprehension/answers'"));
  assert.ok(loader.includes('Warum – und warum nicht anders?'));
  assert.ok(loader.includes('type="radio"'));
  assert.ok(loader.includes('fieldset'));
  assert.ok(loader.includes('required'));
});

test('a delayed previous-topic response cannot overwrite the new page', async () => {
  let resolve;
  const pending = new Promise(r=>{resolve=r});
  const context=sandbox(()=>pending), target=container();
  const done=context.loadTopicChecks('old',target);
  target.isConnected=false;
  resolve({topic_id:'old',questions:[publicQuestion('old')]});
  await done;
  assert.equal(target.innerHTML,'loading');
});

test('topic loader rejects cross-topic questions and escapes content', async () => {
  const wrong=container();
  await sandbox(async()=>({topic_id:'current',questions:[publicQuestion('other')]})).loadTopicChecks('current',wrong);
  assert.ok(wrong.innerHTML.includes('passen nicht'));
  assert.ok(!wrong.innerHTML.includes('<form'));
  let requested;
  const right=container();
  await sandbox(async url=>{requested=url;return {topic_id:'current',questions:[publicQuestion()]}}).loadTopicChecks('current',right);
  assert.equal(requested,'/api/comprehension?topic=current');
  assert.ok(right.innerHTML.includes('&lt;img'));
  assert.ok(!right.innerHTML.includes('<img'));
  assert.ok(right.innerHTML.includes('Richtig'));
  assert.ok(right.innerHTML.includes('Falsch'));
});

test('request failure does not replace the learning content and has a retry', async () => {
  const target=container();
  await sandbox(async()=>{throw new Error('<unavailable>')}).loadTopicChecks('current',target);
  assert.ok(target.innerHTML.includes('&lt;unavailable&gt;'));
  assert.ok(target.innerHTML.includes('Erneut laden'));
});

test('both correct and incorrect answers show server explanation and can be retried', async () => {
  for (const correct of [true,false]) {
    const handlers={}, retryHandlers={}, requests=[];
    const selected={value:'false',focus(){this.focused=true}};
    const button={disabled:false}, fieldset={disabled:false};
    const resultElement={focus(){this.focused=true}};
    const feedback={innerHTML:'',replaceChildren(){this.innerHTML=''},querySelector(selector){
      return selector==='.check-retry'?{addEventListener(name,handler){retryHandlers[name]=handler}}:resultElement;
    }};
    const form={isConnected:true,dataset:{},addEventListener(name,handler){handlers[name]=handler},reset(){this.didReset=true},querySelector(selector){
      return selector==='input:checked'||selector==='input'?selected:selector==='fieldset'?fieldset:selector==='.topic-check-feedback'?feedback:button;
    }};
    const target=container();target.querySelectorAll=()=>[form];
    const context=sandbox(async(url,options)=>{
      requests.push({url,options});
      if(!options)return {topic_id:'current',questions:[publicQuestion()]};
      return {correct,correct_option_ids:[correct?'false':'true'],explanation:'<Why this is correct; why the opposite is not>'};
    });
    await context.loadTopicChecks('current',target);
    await handlers.submit({preventDefault(){}});
    assert.deepEqual(JSON.parse(requests[1].options.body),{topic_id:'current',question_id:'question',response:{option_ids:['false']}});
    assert.equal(requests[1].url,'/api/comprehension/answers');
    assert.ok(feedback.innerHTML.includes(`Deine Antwort ist ${correct?'richtig':'falsch'}`));
    assert.ok(feedback.innerHTML.includes('&lt;Why'));
    assert.ok(feedback.innerHTML.includes('Warum – und warum nicht anders?'));
    assert.equal(resultElement.focused,true);
    assert.equal(fieldset.disabled,true);
    assert.equal(button.disabled,true);
    retryHandlers.click();
    assert.equal(form.didReset,true);
    assert.equal(fieldset.disabled,false);
    assert.equal(button.disabled,false);
    assert.equal(feedback.innerHTML,'');
    assert.equal(form.dataset.answered,undefined);
    assert.equal(selected.focused,true);
  }
});
