pragma solidity ^0.4.24;

//应用管理智能合约
contract AppManager {

    mapping(string=>address) _appIdToAppCreatorAddr;
    mapping(string=>address[]) _appIdToCandidateAddrs;
    mapping(string=>uint8) _appIdToConsensusType;

    address _owner;
    mapping (address => bool) public _isAdmin;

    event LogAddAdmin(address whoAdded, address newAdmin);
    event LogRemoveAdmin(address whoRemoved, address admin);
    event OwnershipTransferred(address indexed previousOwner, address indexed newOwner);

    //权限控制
    modifier onlyOwner {
        require(msg.sender == _owner, "error: op must by admin");
        _;
    }

    modifier onlyAdmin {
        require (_isAdmin[msg.sender]);
        _;
    }

    constructor() public {
        _owner = msg.sender;
        emit OwnershipTransferred(address(0), _owner);
        _isAdmin[_owner] = true;
        emit LogAddAdmin(msg.sender, msg.sender);
    }

    function transferOwnership(address newOwner) public onlyOwner {
        _transferOwnership(newOwner);
    }

    function _transferOwnership(address newOwner) internal {
        require(newOwner != address(0), "Ownable: new owner is the zero address");
        emit OwnershipTransferred(_owner, newOwner);
        _owner = newOwner;
    }

    function addAdmin(address admin) public onlyOwner returns (bool) {
        if(_isAdmin[admin] == false) {
            _isAdmin[admin] = true;
            emit LogAddAdmin(msg.sender, admin);
        }
        return true;
    }

    function removeAdmin(address admin) public onlyOwner returns (bool) {
        if(_isAdmin[admin] == true) {
            _isAdmin[admin] = false;
            emit LogRemoveAdmin(msg.sender, admin);
        }
        return true;
    }

    function setAppCreatorAddr(string appId, address addr) public onlyAdmin {
        _appIdToAppCreatorAddr[appId] = addr;
    }

    function getAppCreatorAddr(string appId) public view returns (address) {
        return _appIdToAppCreatorAddr[appId];
    }

    function setAppCandidateAddrs(string appId, address[] addrs) public onlyAdmin {
        _appIdToCandidateAddrs[appId] = addrs;
    }

    function getAppCandidateAddrs(string appId) public view returns (address[]) {
        return _appIdToCandidateAddrs[appId];
    }

    function setAppConsensusType(string appId, uint8 consensusType) public onlyAdmin {
        _appIdToConsensusType[appId] = consensusType;
    }

    function getAppConsensusType(string appId) public view returns (uint8) {
        return _appIdToConsensusType[appId];
    }
}
