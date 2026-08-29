// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

/// @title ERC20Token
/// @notice Mintable, burnable, pausable ERC-20 token with a single owner.
///         Deployed by the bot service on behalf of the wallet that
///         requested creation; that wallet becomes the owner.
contract ERC20Token {
    string public name;
    string public symbol;
    uint8 public immutable decimals;

    uint256 public totalSupply;
    address public owner;
    bool public paused;

    mapping(address => uint256) public balanceOf;
    mapping(address => mapping(address => uint256)) public allowance;

    event Transfer(address indexed from, address indexed to, uint256 value);
    event Approval(address indexed owner, address indexed spender, uint256 value);
    event OwnershipTransferred(address indexed previousOwner, address indexed newOwner);
    event Mint(address indexed to, uint256 value);
    event Burn(address indexed from, uint256 value);
    event Paused(address account);
    event Unpaused(address account);

    modifier onlyOwner() {
        require(msg.sender == owner, "ERC20Token: caller is not the owner");
        _;
    }

    modifier whenNotPaused() {
        require(!paused, "ERC20Token: token transfers paused");
        _;
    }

    constructor(
        string memory _name,
        string memory _symbol,
        uint8 _decimals,
        uint256 _initialSupply,
        address _owner
    ) {
        require(_owner != address(0), "ERC20Token: owner is zero address");
        name = _name;
        symbol = _symbol;
        decimals = _decimals;
        owner = _owner;

        if (_initialSupply > 0) {
            totalSupply = _initialSupply;
            balanceOf[_owner] = _initialSupply;
            emit Transfer(address(0), _owner, _initialSupply);
        }

        emit OwnershipTransferred(address(0), _owner);
    }

    function transfer(address to, uint256 value) external whenNotPaused returns (bool) {
        _transfer(msg.sender, to, value);
        return true;
    }

    function approve(address spender, uint256 value) external returns (bool) {
        allowance[msg.sender][spender] = value;
        emit Approval(msg.sender, spender, value);
        return true;
    }

    function transferFrom(address from, address to, uint256 value) external whenNotPaused returns (bool) {
        uint256 allowed = allowance[from][msg.sender];
        require(allowed >= value, "ERC20Token: insufficient allowance");
        if (allowed != type(uint256).max) {
            allowance[from][msg.sender] = allowed - value;
        }
        _transfer(from, to, value);
        return true;
    }

    function _transfer(address from, address to, uint256 value) internal {
        require(to != address(0), "ERC20Token: transfer to zero address");
        uint256 fromBalance = balanceOf[from];
        require(fromBalance >= value, "ERC20Token: insufficient balance");
        unchecked {
            balanceOf[from] = fromBalance - value;
        }
        balanceOf[to] += value;
        emit Transfer(from, to, value);
    }

    function mint(address to, uint256 value) external onlyOwner {
        require(to != address(0), "ERC20Token: mint to zero address");
        totalSupply += value;
        balanceOf[to] += value;
        emit Transfer(address(0), to, value);
        emit Mint(to, value);
    }

    function burn(uint256 value) external {
        uint256 bal = balanceOf[msg.sender];
        require(bal >= value, "ERC20Token: burn amount exceeds balance");
        unchecked {
            balanceOf[msg.sender] = bal - value;
        }
        totalSupply -= value;
        emit Transfer(msg.sender, address(0), value);
        emit Burn(msg.sender, value);
    }

    function transferOwnership(address newOwner) external onlyOwner {
        require(newOwner != address(0), "ERC20Token: new owner is zero address");
        address previous = owner;
        owner = newOwner;
        emit OwnershipTransferred(previous, newOwner);
    }

    function pause() external onlyOwner {
        require(!paused, "ERC20Token: already paused");
        paused = true;
        emit Paused(msg.sender);
    }

    function unpause() external onlyOwner {
        require(paused, "ERC20Token: not paused");
        paused = false;
        emit Unpaused(msg.sender);
    }
}
