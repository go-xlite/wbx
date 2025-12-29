var T=window.location.pathname.split("/").filter((q)=>q),x=T.length>0?"/"+T[0]:"";class B{constructor(){let q=window.location.pathname.split("/").filter((J)=>J),G=q.length>0?"/"+q[0]:"";this.baseUrl=`${G}/trail`,this.loading=!1,this.error=null}async fetchServerList(){this.loading=!0,this.error=null;try{let q=await fetch(`${this.baseUrl}/servers/a/list`);if(!q.ok)throw Error(`HTTP error! status: ${q.status}`);let G=await q.json();return this.loading=!1,G}catch(q){throw this.loading=!1,this.error=q.message,q}}async fetchServerDetails(q){this.loading=!0,this.error=null;try{let G=await fetch(`${this.baseUrl}/servers/i/${q}/details`);if(!G.ok)throw Error(`HTTP error! status: ${G.status}`);let J=await G.json();return this.loading=!1,J}catch(G){throw this.loading=!1,this.error=G.message,G}}async fetchFilters(){this.loading=!0,this.error=null;try{let q=await fetch(`${this.baseUrl}/servers/a/filters`);if(!q.ok)throw Error(`HTTP error! status: ${q.status}`);let G=await q.json();return this.loading=!1,G}catch(q){throw this.loading=!1,this.error=q.message,q}}}function w(){return new B}var R=null,$=[],L=[],Y=1,_=20,I=null,W={search:"",region:"",zone:"",state:"",type:""},E={};function K(q){return q.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;").replace(/'/g,"&#039;")}function j(q){let G={};for(let J in E)G[J]=q[E[J]];return G}function V(){document.getElementById("totalServers").textContent=$.length,document.getElementById("displayedServers").textContent=L.length}function O(q){return{running:'<span class="status-badge status-running">● Running</span>',stopped:'<span class="status-badge status-stopped">● Stopped</span>',pending:'<span class="status-badge status-pending">● Pending</span>',stopping:'<span class="status-badge status-stopping">● Stopping</span>',terminated:'<span class="status-badge status-terminated">● Terminated</span>'}[q]||`<span class="status-badge">${K(q)}</span>`}function u(q){if(!q.CPUCores||!q.RAMTotalGB)return"-";return`${q.CPUCores} vCPU, ${q.RAMTotalGB} GB RAM`}function M(q){if(!q)return"-";return new Date(q).toLocaleDateString("en-US",{month:"short",day:"numeric",year:"numeric"})}function y(q){let G=(J)=>typeof q==="object"&&!Array.isArray(q)?q[J]:q[E[J]];return`
        <tr class="clickable-row" data-instance-id="${K(G("ID"))}">
            <td><span class="instance-id">${K(G("ID"))}</span></td>
            <td><span class="instance-name">${K(G("Hostname"))}</span></td>
            <td>${O(G("State"))}</td>
            <td><span class="instance-type">${G("CPUCores")} vCPU, ${G("RAMTotalGB")} GB RAM</span></td>
            <td><span class="instance-ip">${K(G("PublicIPv4")||"-")}</span></td>
            <td><span class="instance-region">${K(G("Region")||"-")}</span></td>
            <td><span class="instance-date">${M(G("LaunchedAt"))}</span></td>
        </tr>
    `}function S(){let q=Math.ceil(L.length/_),G=document.getElementById("pagination");if(q<=1){G.innerHTML="";return}let J="";if(Y>1)J+='<button onclick="window.goToPage('+(Y-1)+')">← Previous</button>';let Q=7,X=Math.max(1,Y-Math.floor(Q/2)),U=Math.min(q,X+Q-1);if(U-X<Q-1)X=Math.max(1,U-Q+1);if(X>1){if(J+='<button onclick="window.goToPage(1)">1</button>',X>2)J+="<span>...</span>"}for(let N=X;N<=U;N++)J+='<button class="'+(N===Y?"active":"")+'" onclick="window.goToPage('+N+')">'+N+"</button>";if(U<q){if(U<q-1)J+="<span>...</span>";J+='<button onclick="window.goToPage('+q+')">'+q+"</button>"}if(Y<q)J+='<button onclick="window.goToPage('+(Y+1)+')">Next →</button>';G.innerHTML=J}function A(){let q=document.getElementById("instancesTableBody"),G=(Y-1)*_,J=G+_,Q=L.slice(G,J);if(Q.length===0){q.innerHTML='<tr class="empty-row"><td colspan="7" class="empty-state">No instances found</td></tr>';return}q.innerHTML=Q.map(y).join(""),S(),h()}function f(q){W.search=q||"",Z()}function Z(){L=$.filter((q)=>{let G=Array.isArray(q)?j(q):q;if(W.search){let J=W.search.toLowerCase();if(!(G.ID.toLowerCase().includes(J)||G.Hostname.toLowerCase().includes(J)||G.PublicIPv4&&G.PublicIPv4.toLowerCase().includes(J)||G.PrivateIPv4&&G.PrivateIPv4.toLowerCase().includes(J)))return!1}if(W.region&&G.Region!==W.region)return!1;if(W.zone&&G.Zone!==W.zone)return!1;if(W.state&&G.State!==W.state)return!1;if(W.type){if(u(G)!==W.type)return!1}return!0}),Y=1,V(),A()}function b(q){I=q;let G=document.getElementById("regionFilter");q.regions.forEach((U)=>{let N=document.createElement("option");N.value=U,N.textContent=U,G.appendChild(N)});let J=document.getElementById("zoneFilter");q.zones.forEach((U)=>{let N=document.createElement("option");N.value=U,N.textContent=U,J.appendChild(N)});let Q=document.getElementById("stateFilter");q.states.forEach((U)=>{let N=document.createElement("option");N.value=U,N.textContent=U.charAt(0).toUpperCase()+U.slice(1),Q.appendChild(N)});let X=document.getElementById("typeFilter");q.instanceTypes.forEach((U)=>{let N=document.createElement("option");N.value=U,N.textContent=U,X.appendChild(N)})}function h(){document.querySelectorAll(".clickable-row").forEach((G)=>{G.addEventListener("click",async()=>{let J=G.getAttribute("data-instance-id");await F(J)})})}async function F(q){let G=document.getElementById("detailsModal"),J=document.getElementById("modalContent");G.style.display="flex",J.innerHTML='<div class="modal-loader">Loading instance details...</div>';try{let Q=await R.fetchServerDetails(q);J.innerHTML=k(Q)}catch(Q){J.innerHTML=`<div class="error-state">Failed to load details: ${K(Q.message)}</div>`}}function k(q){let G=`
        <div class="details-header">
            <h2>${K(q.Hostname)}</h2>
            <div class="details-id">${K(q.ID)}</div>
        </div>
        
        <div class="details-section">
            <h3>Instance Information</h3>
            <div class="details-grid">
                <div class="detail-item"><strong>Status:</strong> ${O(q.State)}</div>
                <div class="detail-item"><strong>Region:</strong> ${K(q.Region||"-")}</div>
                <div class="detail-item"><strong>Zone:</strong> ${K(q.Zone||"-")}</div>
                <div class="detail-item"><strong>OS:</strong> ${K(q.OS||"-")}</div>
                <div class="detail-item"><strong>Launched:</strong> ${M(q.LaunchedAt)}</div>
                <div class="detail-item"><strong>Uptime:</strong> ${K(q.Uptime||"-")}</div>
            </div>
        </div>
    `;if(q.ServerInfo)G+=`
            <div class="details-section">
                <h3>Server Hardware</h3>
                <div class="details-grid">
                    <div class="detail-item"><strong>Brand:</strong> ${K(q.ServerInfo.Brand)}</div>
                    <div class="detail-item"><strong>Model:</strong> ${K(q.ServerInfo.Model)}</div>
                    <div class="detail-item"><strong>Serial Number:</strong> ${K(q.ServerInfo.SerialNumber)}</div>
                    <div class="detail-item"><strong>Manufacture Year:</strong> ${q.ServerInfo.ManufactureYear}</div>
                    <div class="detail-item"><strong>Warranty Expiry:</strong> ${K(q.ServerInfo.WarrantyExpiry)}</div>
                    <div class="detail-item"><strong>Datacenter:</strong> ${K(q.ServerInfo.Datacenter)}</div>
                    <div class="detail-item"><strong>Rack:</strong> ${K(q.ServerInfo.Rack)}</div>
                    <div class="detail-item"><strong>Position:</strong> ${q.ServerInfo.Position}</div>
                </div>
            </div>
        `;if(q.CPUInfo)G+=`
            <div class="details-section">
                <h3>CPU Information</h3>
                <div class="details-grid">
                    <div class="detail-item"><strong>Brand:</strong> ${K(q.CPUInfo.Brand)}</div>
                    <div class="detail-item"><strong>Model:</strong> ${K(q.CPUInfo.Model)}</div>
                    <div class="detail-item"><strong>Cores:</strong> ${q.CPUInfo.Cores}</div>
                    <div class="detail-item"><strong>Threads:</strong> ${q.CPUInfo.Threads}</div>
                    <div class="detail-item"><strong>Speed:</strong> ${q.CPUInfo.SpeedGHz} GHz</div>
                    <div class="detail-item"><strong>Cache Size:</strong> ${q.CPUInfo.CacheSize} MB</div>
                    <div class="detail-item"><strong>Socket Count:</strong> ${q.CPUInfo.SocketCount}</div>
                </div>
            </div>
        `;if(q.RAMInfo)G+=`
            <div class="details-section">
                <h3>Memory (RAM)</h3>
                <div class="details-grid">
                    <div class="detail-item"><strong>Total:</strong> ${q.RAMInfo.TotalGB} GB</div>
                    <div class="detail-item"><strong>Configuration:</strong> ${K(q.RAMInfo.Configuration)}</div>
                    <div class="detail-item"><strong>Type:</strong> ${K(q.RAMInfo.Type)}</div>
                    <div class="detail-item"><strong>Speed:</strong> ${q.RAMInfo.Speed} MHz</div>
                    <div class="detail-item"><strong>ECC:</strong> ${q.RAMInfo.ECC?"Yes":"No"}</div>
                    <div class="detail-item"><strong>Manufacturer:</strong> ${K(q.RAMInfo.Manufacturer)}</div>
                </div>
            </div>
        `;if(q.StorageDisks&&q.StorageDisks.length>0)G+=`
            <div class="details-section">
                <h3>Storage Disks (${q.StorageDisks.length})</h3>
                <div class="storage-table">
                    <table>
                        <thead>
                            <tr>
                                <th>Slot</th>
                                <th>Type</th>
                                <th>Brand/Model</th>
                                <th>Capacity</th>
                                <th>Used</th>
                                <th>Usage</th>
                                <th>Health</th>
                                <th>Temp</th>
                            </tr>
                        </thead>
                        <tbody>
        `,q.StorageDisks.forEach((J)=>{let Q=J.HealthStatus==="healthy"?"health-ok":"health-warning";G+=`
                <tr>
                    <td>${J.Slot}</td>
                    <td><span class="disk-type">${K(J.Type)}</span></td>
                    <td><div class="disk-model">${K(J.Brand)} ${K(J.Model)}</div><div class="disk-serial">${K(J.SerialNumber)}</div></td>
                    <td>${J.CapacityGB} GB</td>
                    <td>${J.UsedGB} GB</td>
                    <td><div class="usage-bar"><div class="usage-fill" style="width: ${J.UsagePercent}%"></div></div><span class="usage-text">${J.UsagePercent.toFixed(1)}%</span></td>
                    <td><span class="${Q}">${K(J.HealthStatus)}</span></td>
                    <td>${J.TemperatureC}°C</td>
                </tr>
            `}),G+=`
                        </tbody>
                    </table>
                </div>
            </div>
        `;if(q.NetworkNICs&&q.NetworkNICs.length>0)G+=`
            <div class="details-section">
                <h3>Network Interfaces (${q.NetworkNICs.length})</h3>
                <div class="network-table">
                    <table>
                        <thead>
                            <tr>
                                <th>Interface</th>
                                <th>Vendor/Model</th>
                                <th>IPv4</th>
                                <th>IPv6</th>
                                <th>MAC Address</th>
                                <th>Bandwidth</th>
                                <th>Status</th>
                            </tr>
                        </thead>
                        <tbody>
        `,q.NetworkNICs.forEach((J)=>{let Q=J.Status==="up"?"status-running":"status-stopped";G+=`
                <tr>
                    <td><strong>${K(J.Interface)}</strong></td>
                    <td><div class="nic-vendor">${K(J.Vendor)}</div><div class="nic-model">${K(J.Model)}</div></td>
                    <td>${K(J.IPv4)}</td>
                    <td class="ipv6">${K(J.IPv6)}</td>
                    <td class="mac-address">${K(J.MACAddress)}</td>
                    <td>${J.BandwidthGbps} Gbps</td>
                    <td><span class="status-badge ${Q}">● ${K(J.Status)}</span></td>
                </tr>
            `}),G+=`
                        </tbody>
                    </table>
                </div>
            </div>
        `;return G}function H(){let q=document.getElementById("detailsModal");q.style.display="none"}function D(q){let G=document.getElementById("refreshBtn"),J=document.getElementById("loader"),Q=document.querySelector(".table-container");if(q){if(G.disabled=!0,J.style.display="block",Q)Q.style.opacity="0.5"}else if(G.disabled=!1,J.style.display="none",Q)Q.style.opacity="1"}function P(q){let G=document.getElementById("instancesTableBody");G.innerHTML=`<tr class="empty-row"><td colspan="7"><div class="error-state">❌ Error: ${K(q)}</div></td></tr>`}async function C(){D(!0);try{let q=await R.fetchServerList();if(q.columns&&q.data)E={},q.columns.forEach((J,Q)=>{E[J]=Q}),$=q.data;else $=q||[];L=[...$],V(),A();let G=new Date().toLocaleTimeString();if(document.getElementById("lastUpdate").textContent=G,!I){let J=await R.fetchFilters();b(J)}}catch(q){console.error("Failed to load servers:",q),P(q.message)}finally{D(!1)}}window.goToPage=function(q){Y=q,A(),window.scrollTo({top:0,behavior:"smooth"})};function z(){R=w();let q=document.getElementById("refreshBtn");if(q)q.addEventListener("click",C);let G=document.getElementById("searchInput");if(G)G.addEventListener("input",(N)=>{f(N.target.value)});let J=document.getElementById("regionFilter");if(J)J.addEventListener("change",(N)=>{W.region=N.target.value,Z()});let Q=document.getElementById("zoneFilter");if(Q)Q.addEventListener("change",(N)=>{W.zone=N.target.value,Z()});let X=document.getElementById("stateFilter");if(X)X.addEventListener("change",(N)=>{W.state=N.target.value,Z()});let U=document.getElementById("typeFilter");if(U)U.addEventListener("change",(N)=>{W.type=N.target.value,Z()});C()}if(document.readyState==="loading")document.addEventListener("DOMContentLoaded",z);else z();window.closeModal=H;
